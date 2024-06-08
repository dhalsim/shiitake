package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"strings"

	"fiatjaf.com/shiitake/components/icon_placeholder"
	"fiatjaf.com/shiitake/global"
	"github.com/diamondburned/adaptive"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/app"
	"github.com/diamondburned/gotkit/app/locale"
	"github.com/diamondburned/gotkit/components/autoscroll"
	"github.com/diamondburned/gotkit/gtkutil"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/nbd-wtf/go-nostr/nip29"
)

type messageRow struct {
	*gtk.ListBoxRow
	message *Message
	event   *nostr.Event
}

type MessagesView struct {
	*adaptive.LoadablePage

	LoadMore *gtk.Button
	Composer *ComposerView

	listStack *gtk.Stack

	currentGroup *global.Group
	switchTo     func(gad nip29.GroupAddress)

	msgs map[string]messageRow

	state struct {
		row      *gtk.ListBoxRow
		replying bool
	}

	ctx context.Context
	gad nip29.GroupAddress
}

const (
	loadMoreBatch = 25 // load this many more messages on scroll
	initialBatch  = 15 // load this many messages on startup
	idealMaxCount = 50 // ideally keep this many messages in the view
)

func NewMessagesView(ctx context.Context) *MessagesView {
	v := &MessagesView{
		msgs: make(map[string]messageRow),
		ctx:  ctx,
	}

	var me *global.Me

	v.listStack = gtk.NewStack()
	v.listStack.SetTransitionType(gtk.StackTransitionTypeCrossfade)

	plc := icon_placeholder.New("chat-bubbles-empty-symbolic")

	var current nip29.GroupAddress

	composerOverlay := gtk.NewOverlay()

	joinButton := gtk.NewButtonWithLabel("Join")
	joinButton.SetHExpand(true)
	joinButton.SetHAlign(gtk.AlignFill)
	joinButton.AddCSSClass("p-8")
	joinButton.AddCSSClass("mx-4")
	joinButton.AddCSSClass("my-2")
	joinButton.AddCSSClass("suggested-action")
	joinButton.SetTooltipText("Join Group")
	joinButton.ConnectClicked(func() {
		joinButton.SetLabel("Joining...")
		joinButton.SetSensitive(false)
		joinButton.RemoveCSSClass("suggested-action")

		go func() {
			if err := global.JoinGroup(ctx, current); err != nil {
				win.ErrorToast(err.Error())
			}

			joinButton.SetLabel("Join")
			joinButton.SetSensitive(true)
			joinButton.AddCSSClass("suggested-action")
		}()
	})

	bottomStack := gtk.NewStack()
	bottomStack.AddChild(composerOverlay)
	bottomStack.AddChild(joinButton)
	bottomStack.SetVisibleChild(joinButton)

	outerBox := gtk.NewBox(gtk.OrientationVertical, 0)
	outerBox.SetHExpand(true)
	outerBox.SetVExpand(true)
	outerBox.Append(v.listStack)
	outerBox.Append(bottomStack)

	v.LoadablePage = adaptive.NewLoadablePage()
	v.LoadablePage.SetTransitionDuration(125)
	v.LoadablePage.SetChild(outerBox)

	// If the window gains focus, try to carefully mark the channel as read.
	var windowSignal glib.SignalHandle
	v.ConnectMap(func() {
		window := app.GTKWindowFromContext(ctx)
		windowSignal = window.NotifyProperty("is-active", func() {
			if v.IsActive() {
				v.MarkRead()
			}
		})
	})
	// Immediately disconnect the signal when the widget is unmapped.
	// This should prevent v from being referenced forever.
	v.ConnectUnmap(func() {
		window := app.GTKWindowFromContext(ctx)
		window.HandlerDisconnect(windowSignal)
		windowSignal = 0
	})

	v.switchTo = func(gad nip29.GroupAddress) {
		me = global.GetMe(ctx)

		if current.Equals(gad) {
			return
		}
		current = gad

		if !gad.IsValid() {
			// empty, switch to placeholder
			v.LoadablePage.SetChild(plc)
			return
		}

		// otherwise we have something,
		// so switch back to the main thing which is outerBox
		v.LoadablePage.SetChild(outerBox)

		gtkutil.NotifyProperty(v.Parent(), "transition-running", func() bool {
			if !v.LoadablePage.Stack.TransitionRunning() {
				return true
			}
			return false
		})

		group := global.GetGroup(ctx, gad)
		win.main.Header.SetTitleWidget(adw.NewWindowTitle(group.Name, group.Address.String()))
		v.currentGroup = group

		// get existing group messages view (scroll) and list
		var scroll *gtk.ScrolledWindow
		var list *gtk.ListBox

		if scrollI := v.listStack.ChildByName(gad.String()); scrollI != nil {
			scroll, _ = scrollI.(*gtk.ScrolledWindow)

			// fragile: this is depends on the internal structure of autoscroll.Window
			switch child := scroll.Child().(type) {
			case *gtk.Box:
				list = child.LastChild().(*gtk.ListBox)
			case *gtk.Viewport:
				list = child.Child().(*gtk.Box).LastChild().(*gtk.ListBox)
			default:
				panic("unexpected type")
			}

		} else {
			// create list if we haven't done that before
			// TODO: we need a context here or something so the subscription is canceled if this group is removed
			list = gtk.NewListBox()
			list.SetSelectionMode(gtk.SelectionNone)

			loadMore := gtk.NewButton()
			loadMore.SetLabel("Show More")
			loadMore.SetHExpand(true)
			loadMore.SetSensitive(true)
			loadMore.ConnectClicked(v.loadMore)
			loadMore.Hide()

			clampBox := gtk.NewBox(gtk.OrientationVertical, 0)
			clampBox.SetHExpand(true)
			clampBox.SetVExpand(true)
			clampBox.SetVAlign(gtk.AlignEnd)
			clampBox.Append(loadMore)
			clampBox.Append(list)

			scrollW := autoscroll.NewWindow()
			scrollW.SetVExpand(true)
			scrollW.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
			scrollW.SetPropagateNaturalWidth(true)
			scrollW.SetPropagateNaturalHeight(true)
			scrollW.SetChild(clampBox)
			scrollW.OnBottomed(v.onScrollBottomed)
			scroll = scrollW.ScrolledWindow

			scrollAdjustment := scroll.VAdjustment()
			scrollAdjustment.ConnectValueChanged(func() {
				// Replicate adw.ToolbarView's behavior: if the user scrolls up, then
				// show a small drop shadow at the bottom of the view. We're not using
				// the actual widget, because it adds a WindowHandle at the bottom,
				// which breaks double-clicking.
				value := scrollAdjustment.Value()
				upper := scrollAdjustment.Upper()
				psize := scrollAdjustment.PageSize()
				if value < upper-psize {
				} else {
					scroll.RemoveCSSClass("undershoot-bottom")
				}
			})

			vp := scrollW.Viewport()
			vp.SetScrollToFocus(true)

			lastAppendedAuthor := ""
			upsertMessage := func(event *nostr.Event, pos int) {
				id := event.ID
				if _, ok := v.msgs[id]; ok {
					return
				}

				authorIdem := false
				if pos == -1 {
					if event.PubKey == lastAppendedAuthor {
						authorIdem = true
					} else {
						lastAppendedAuthor = event.PubKey
					}
				}

				cmessage := NewMessage(v.ctx, event, event.PubKey == me.PubKey, authorIdem)
				row := gtk.NewListBoxRow()
				row.SetName(id)
				row.SetChild(cmessage)
				msgRow := messageRow{
					ListBoxRow: row,
					message:    cmessage,
					event:      event,
				}

				v.msgs[id] = msgRow

				list.Insert(row, pos)
				list.Display().Flush()
				list.SetFocusChild(row)
			}

			showingLoadMoreAlready := false

			// insert previously loaded messages
			gtkutil.Async(v.ctx, func() func() {
				<-group.EOSE

				for _, evt := range group.Messages {
					upsertMessage(evt, -1)
				}

				if scroll.AllocatedHeight() < int(vp.VAdjustment().Upper()) {
					loadMore.Show()
				}

				return func() {
					scrollW.ScrollToBottom()
				}
			})

			// listen for new messages
			go func() {
				for evt := range group.NewMessage {
					glib.IdleAdd(func() {
						upsertMessage(evt, -1)

						if !showingLoadMoreAlready &&
							scroll.AllocatedHeight() < int(vp.VAdjustment().Upper()) {
							showingLoadMoreAlready = true
							loadMore.Show()
						}
					})
				}
			}()

			// insert in the stack
			v.listStack.AddNamed(scroll, gad.String())
		}

		// make it visible
		v.listStack.SetVisibleChild(scroll)

		// check if we should be a member of this group
		<-me.ListLoaded
		if me.InGroup(gad) {
			// create composer and forward typing
			v.Composer = NewComposerView(ctx, v, group)
			composerOverlay.SetChild(v.Composer)
			gtkutil.ForwardTyping(list, v.Composer.Input)
			bottomStack.SetVisibleChild(composerOverlay)
		} else {
			bottomStack.SetVisibleChild(joinButton)
		}
	}

	v.LoadablePage.SetLoading()

	return v
}

func (v *MessagesView) visibleList() *gtk.ListBox {
	listI := v.listStack.VisibleChild()
	if listI == nil {
		return nil
	}
	return listI.(*gtk.ListBox)
}

// HeaderButtons returns the header buttons widget for the message view.
// This widget is kept on the header bar for as long as the message view is
// active.
func (v *MessagesView) HeaderButtons() []gtk.Widgetter {
	var buttons []gtk.Widgetter

	// if v.guildID.IsValid() {
	// 	summariesButton.SetIconName("speaker-notes-symbolic")
	// 	summariesButton.SetTooltipText(locale.Get("Message Summaries"))
	// 	buttons = append(buttons, summariesButton)

	// 	state := gtkcord.FromContext(v.ctx)
	// 	if len(state.SummaryState.Summaries(v.chID)) == 0 {
	// 		summariesButton.SetSensitive(false)
	// 		var unbind func()
	// 		unbind = state.AddHandlerForWidget(summariesButton, func(ev *gateway.ConversationSummaryUpdateEvent) {
	// 			if ev.ChannelID == v.chID && len(ev.Summaries) > 0 {
	// 				summariesButton.SetSensitive(true)
	// 				unbind()
	// 			}
	// 		})
	// 	}

	// 	infoButton.SetIconName("dialog-information-symbolic")
	// 	infoButton.SetTooltipText(locale.Get("Channel Info"))
	// 	buttons = append(buttons, infoButton)
	// }

	return buttons
}

func (v *MessagesView) loadMore() {
	list := v.visibleList()
	if list == nil {
		return
	}

	firstRow, _ := list.FirstChild().(*gtk.ListBoxRow)
	if firstRow == nil {
		return
	}
	msg, ok := v.msgs[firstRow.Name()]
	if !ok {
		return
	}
	firstID := msg.event.ID

	log.Println("loading more messages for", v.gad, firstID)

	ctx := v.ctx
	// state := gtkcord.FromContext(ctx).Online()

	// prevScrollVal := v.Scroll.VAdjustment().Value()
	// prevScrollMax := v.Scroll.VAdjustment().Upper()

	// upsertMessages := func(msgs []nostr.Event) {
	// 	infos := make([]messageInfo, len(msgs))
	// 	for i := range msgs {
	// 		infos[i] = newMessageInfo(&msgs[i])
	// 	}

	// 	for i, msg := range msgs {
	// 		flags := 0 |
	// 			upsertFlagOverrideCollapse |
	// 			upsertFlagPrepend

	// 		// Manually prepend our own messages. This also requires us to
	// 		// manually check if we should collapse the message.
	// 		if i != len(msgs)-1 {
	// 			curr := infos[i]
	// 			last := infos[i+1]
	// 			if shouldBeCollapsed(curr, last) {
	// 				flags |= upsertFlagCollapsed
	// 			}
	// 		}

	// 		w := v.upsertMessage(msg.ID, infos[i], flags)
	// 		w.Update(&gateway.MessageCreateEvent{Message: msg})
	// 	}

	// 	// Do this on the next iteration of the main loop so that the scroll
	// 	// adjustment has time to update.
	// 	glib.IdleAdd(func() {
	// 		// Calculate the offset at which to scroll to after loading more
	// 		// messages.
	// 		currentScrollMax := v.Scroll.VAdjustment().Upper()

	// 		vadj := v.Scroll.VAdjustment()
	// 		vadj.SetValue(prevScrollVal + (currentScrollMax - prevScrollMax))
	// 	})

	// 	// Style the first prepended message to add a visual indicator for the
	// 	// user.
	// 	first := v.msgs[messageKeyID(msgs[0].ID)]

	// 	// Remove this visual indicator after a short while.
	// 	glib.TimeoutSecondsAdd(10, func() {
	// 		first.ListBoxRow.RemoveCSSClass("message-first-prepended")
	// 	})
	// }

	// stateMessages, err := state.Cabinet.Messages(v.chID)
	// if err == nil && len(stateMessages) > 0 {
	// 	// State messages are ordered last first, so we can traverse them.
	// 	var found bool
	// 	for i, m := range stateMessages {
	// 		if m.ID < firstID {
	// 			log.Println("found earlier message in state, content:", m.Content)
	// 			stateMessages = stateMessages[i:]
	// 			found = true
	// 			break
	// 		}
	// 	}

	// 	if found {
	// 		if len(stateMessages) > loadMoreBatch {
	// 			stateMessages = stateMessages[:loadMoreBatch]
	// 		}
	// 		upsertMessages(stateMessages)
	// 		return
	// 	}
	// }

	gtkutil.Async(ctx, func() func() {
		// messages, err := state.MessagesBefore(v.chID, firstID, loadMoreBatch)
		// if err != nil {
		// 	app.Error(ctx, fmt.Errorf("failed to load more messages: %w", err))
		// 	return nil
		// }

		return func() {
			// 	if len(messages) > 0 {
			// 		upsertMessages(messages)
			// 	}

			// 	if len(messages) < loadMoreBatch {
			// 		// We've reached the end of the channel's history.
			// 		// Disable the load more button.
			// 		v.LoadMore.SetSensitive(false)
			// 	}
		}
	})
}

func (v *MessagesView) deleteMessage(list *gtk.ListBox, id string) {
	eachChild(list, func(lbr *gtk.ListBoxRow) bool {
		if lbr.Name() == id {
			list.Remove(lbr)
			return true
		}
		return false
	})
	delete(v.msgs, id)
}

func (v *MessagesView) updateMember(list *gtk.ListBox, pubkey string) {
	eachChild(list, func(lbr *gtk.ListBoxRow) bool {
		fmt.Println("updating member", pubkey)

		// fragile: this depends on the hierarchy of components: message > rightBox > topLabel
		label := lbr.Child().(*gtk.Box).LastChild().(*gtk.Box).FirstChild().(*gtk.Label)
		// fragile: this depends on the string given to the tooltip
		npub := strings.Split(strings.Split(label.TooltipMarkup(), "(")[1], ")")[0]
		_, data, _ := nip19.Decode(npub)
		if pubkey == data.(string) {
			// replace avatar
			// avatar := lbr.Child().(*gtk.Box).FirstChild()
			// lbr.Child().(*gtk.Box).InsertBefore(newAvatar, avatar)
			// lbr.Child().(*gtk.Box).Remove(avatar)

			// replace toplabel
			// TODO
		}
		return false
	})
}

func (v *MessagesView) updateMessageReactions(id string) {
	widget, ok := v.msgs[id]
	if !ok || widget.message == nil {
		return
	}

	// state := gtkcord.FromContext(v.ctx)
	// msg, _ := state.Cabinet.Message(v.chID, id)
	// if msg == nil {
	// 	return
	// }

	// content := widget.message.Content()
	// content.SetReactions(msg.Reactions)
}

func (v *MessagesView) messageIDFromRow(row *gtk.ListBoxRow) string {
	if row == nil {
		return ""
	}

	return row.Name()
}

func (v *MessagesView) ScrollToMessage(gad nip29.GroupAddress, id string) {
	msg, ok := v.msgs[id]
	if !ok {
		slog.Warn(
			"tried to scroll to non-existent message",
			"id", id)
		return
	}

	v.listStack.SetVisibleChildName(gad.String())

	if !msg.ListBoxRow.GrabFocus() {
		slog.Warn(
			"failed to grab focus of message",
			"id", id)
	}
}

// AddReaction adds an reaction to the message with the given ID.
func (v *MessagesView) AddReaction(id string, emoji discord.APIEmoji) {
	// state := gtkcord.FromContext(v.ctx)
	// emoji = discord.APIEmoji(gtkcord.SanitizeEmoji(string(emoji)))

	// gtkutil.Async(v.ctx, func() func() {
	// 	if err := state.React(v.chID, id, emoji); err != nil {
	// 		err = errors.Wrap(err, "Failed to react:")
	// 		return func() {
	// 			toast := adw.NewToast(locale.Get("Cannot react to message"))
	// 			toast.SetTimeout(0)
	// 			toast.SetButtonLabel(locale.Get("Logs"))
	// 			toast.SetActionName("")
	// 		}
	// 	}
	// 	return nil
	// })
}

// ReplyTo starts replying to the message with the given ID.
func (v *MessagesView) ReplyTo(id string) {
	v.stopEditingOrReplying()

	msg, ok := v.msgs[id]
	if !ok || msg.message == nil || msg.message.Event == nil {
		return
	}

	v.state.row = msg.ListBoxRow
	v.state.replying = true

	v.Composer.StartReplyingTo(msg.message.Event)
}

func (v *MessagesView) stopEditingOrReplying() {
	if v.state.row == nil {
		return
	}

	if v.state.replying {
		v.Composer.StopReplying()
		v.state.row.RemoveCSSClass("message-replying")
	}
}

func (v *MessagesView) Delete(id string) {
	username := "?" // juuust in case

	row, ok := v.msgs[id]
	if ok {
		user := global.GetUser(v.ctx, row.message.Event.PubKey)
		username = fmt.Sprintf(`<span weight="normal">%s</span>`, user.ShortName())
	}

	window := app.GTKWindowFromContext(v.ctx)
	dialog := adw.NewMessageDialog(window,
		locale.Get("Delete Message"),
		locale.Sprintf("Are you sure you want to delete %s's message?", username))
	dialog.SetBodyUseMarkup(true)
	dialog.AddResponse("cancel", locale.Get("_Cancel"))
	dialog.AddResponse("delete", locale.Get("_Delete"))
	dialog.SetResponseAppearance("delete", adw.ResponseDestructive)
	dialog.SetDefaultResponse("cancel")
	dialog.SetCloseResponse("cancel")
	dialog.ConnectResponse(func(response string) {
		switch response {
		case "delete":
			v.delete(id)
		}
	})
	dialog.Show()
}

func (v *MessagesView) delete(id string) {
	if msg, ok := v.msgs[id]; ok {
		// Visual indicator.
		msg.SetSensitive(false)
	}
}

func (v *MessagesView) onScrollBottomed() {
	if v.IsActive() {
		v.MarkRead()
	}

	// Try to clean up the top messages.
	// Fast path: check our cache first.
	if len(v.msgs) > idealMaxCount {
		var count int

		list := v.listStack.VisibleChild().(*gtk.ListBox)
		row, _ := list.LastChild().(*gtk.ListBoxRow)
		for row != nil {
			next, _ := row.PrevSibling().(*gtk.ListBoxRow)

			if count < idealMaxCount {
				count++
			} else {
				// Start purging messages.
				list.Remove(row)
				delete(v.msgs, row.Name())
			}

			row = next
		}
	}
}

// MarkRead marks the view's latest messages as read.
func (v *MessagesView) MarkRead() {
	// state := gtkcord.FromContext(v.ctx)
	// Grab the last message from the state cache, since we sometimes don't even
	// render blocked messages.
	// msgs, _ := state.Cabinet.Messages(v.ChannelID())
	// if len(msgs) == 0 {
	// 	return
	// }

	// state.ReadState.MarkRead(v.ChannelID(), msgs[0].ID)

	// readState := state.ReadState.ReadState(v.ChannelID())
	// if readState != nil {
	// 	log.Println("message.MessagesView.MarkRead: marked", msgs[0].ID, "as read, last read", readState.LastMessageID)
	// }
}

// IsActive returns true if MessagesView is active and visible. This implies that the
// window is focused.
func (v *MessagesView) IsActive() bool {
	win := app.GTKWindowFromContext(v.ctx)
	return win.IsActive() && v.Mapped()
}
