package main

import (
	"context"
	"fmt"
	"log"

	"fiatjaf.com/shiitake/components/composer"
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
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip29"
	"golang.org/x/exp/slog"
)

type messageRow struct {
	*gtk.ListBoxRow
	message *cozyMessage
	event   *nostr.Event
}

// MessagesView is a message view widget.
type MessagesView struct {
	*adaptive.LoadablePage

	ToastOverlay *adw.ToastOverlay
	LoadMore     *gtk.Button
	Scroll       *autoscroll.Window
	Composer     *composer.View

	listStack *gtk.Stack

	currentGroup *global.Group
	switchTo     func(gad nip29.GroupAddress)

	loggedUser string
	msgs       map[string]messageRow

	state struct {
		row      *gtk.ListBoxRow
		replying bool
	}

	ctx context.Context
	gad nip29.GroupAddress
}

var messagesViewCSS = cssutil.Applier("message-view", `
.message-list {
  background: none;
}
.message-list > row {
  transition: linear 150ms background-color;
  box-shadow: none;
  background: none;
  background-image: none;
  background-color: transparent;
  padding: 0;
  border: 2px solid transparent;
}
.message-list > row:focus,
.message-list > row:hover {
  transition: none;
}
.message-list > row:focus {
  background-color: alpha(@theme_fg_color, 0.125);
}
.message-list > row:hover {
  background-color: alpha(@theme_fg_color, 0.075);
}
.message-list > row.message-replying {
  background-color: alpha(@theme_selected_bg_color, 0.15);
  border-color: alpha(@theme_selected_bg_color, 0.55);
}
.message-list > row.message-sending {
  opacity: 0.65;
}
.message-list > row.message-first-prepended {
  border-bottom: 1.5px dashed alpha(@theme_fg_color, 0.25);
  padding-bottom: 2.5px;
}
.message-show-more {
  background: none;
  border-radius: 0;
  font-size: 0.85em;
  opacity: 0.65;
}
.message-show-more:hover {
  background: alpha(@theme_fg_color, 0.075);
}
.messages-typing-indicator {
  margin-top: -1em;
}
.messages-typing-box {
  background-color: @theme_bg_color;
}
.message-list,
.message-scroll scrollbar.vertical {
  margin-bottom: 1em;
}
`)

const (
	loadMoreBatch = 25 // load this many more messages on scroll
	initialBatch  = 15 // load this many messages on startup
	idealMaxCount = 50 // ideally keep this many messages in the view
)

func applyViewClamp(clamp *adw.Clamp) {
	// clamp.SetMaximumSize(messagesWidth.Value())
	// Set tightening threshold to 90% of the clamp's width.
	// clamp.SetTighteningThreshold(int(float64(messagesWidth.Value()) * 0.9))
}

func NewMessagesView(ctx context.Context) *MessagesView {
	v := &MessagesView{
		msgs: make(map[string]messageRow),
		ctx:  ctx,
	}

	v.listStack = gtk.NewStack()
	v.listStack.SetTransitionType(gtk.StackTransitionTypeCrossfade)

	plc := icon_placeholder.New("chat-bubbles-empty-symbolic")

	v.LoadMore = gtk.NewButton()
	v.LoadMore.AddCSSClass("message-show-more")
	v.LoadMore.SetLabel(locale.Get("Show More"))
	v.LoadMore.SetHExpand(true)
	v.LoadMore.SetSensitive(true)
	v.LoadMore.ConnectClicked(v.loadMore)

	clampBox := gtk.NewBox(gtk.OrientationVertical, 0)
	clampBox.SetHExpand(true)
	clampBox.SetVExpand(true)
	clampBox.Append(v.LoadMore)
	clampBox.Append(v.listStack)

	// Require 2 clamps, one inside the scroll view and another outside the
	// scroll view. This way, the scrollbars will be on the far right rather
	// than being stuck in the middle.
	clampScroll := adw.NewClamp()
	clampScroll.SetChild(clampBox)
	applyViewClamp(clampScroll)

	v.Scroll = autoscroll.NewWindow()
	v.Scroll.AddCSSClass("message-scroll")
	v.Scroll.SetVExpand(true)
	v.Scroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	v.Scroll.SetPropagateNaturalWidth(true)
	v.Scroll.SetPropagateNaturalHeight(true)
	v.Scroll.SetChild(clampScroll)

	v.Scroll.OnBottomed(v.onScrollBottomed)

	scrollAdjustment := v.Scroll.VAdjustment()
	scrollAdjustment.ConnectValueChanged(func() {
		// Replicate adw.ToolbarView's behavior: if the user scrolls up, then
		// show a small drop shadow at the bottom of the view. We're not using
		// the actual widget, because it adds a WindowHandle at the bottom,
		// which breaks double-clicking.
		value := scrollAdjustment.Value()
		upper := scrollAdjustment.Upper()
		psize := scrollAdjustment.PageSize()
		if value < upper-psize {
			v.Scroll.AddCSSClass("undershoot-bottom")
		} else {
			v.Scroll.RemoveCSSClass("undershoot-bottom")
		}
	})

	vp := v.Scroll.Viewport()
	vp.SetScrollToFocus(true)

	v.Composer = composer.NewView(ctx, v, nip29.GroupAddress{})

	composerOverlay := gtk.NewOverlay()
	composerOverlay.SetChild(v.Composer)

	composerClamp := adw.NewClamp()
	composerClamp.SetChild(composerOverlay)
	applyViewClamp(composerClamp)

	outerBox := gtk.NewBox(gtk.OrientationVertical, 0)
	outerBox.SetHExpand(true)
	outerBox.SetVExpand(true)
	outerBox.Append(v.Scroll)
	outerBox.Append(composerClamp)

	v.ToastOverlay = adw.NewToastOverlay()
	v.ToastOverlay.SetChild(outerBox)

	v.LoadablePage = adaptive.NewLoadablePage()
	v.LoadablePage.SetTransitionDuration(125)
	v.LoadablePage.SetChild(v.ToastOverlay)

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

	var current nip29.GroupAddress
	v.switchTo = func(gad nip29.GroupAddress) {
		if current.Equals(gad) {
			return
		}

		if !gad.IsValid() {
			// empty, switch to placeholder
			v.ToastOverlay.SetChild(plc)
			return
		}

		current = gad
		// otherwise we have something,
		// so switch back to the main thing which is outerBox
		v.ToastOverlay.SetChild(outerBox)

		gtkutil.NotifyProperty(v.Parent(), "transition-running", func() bool {
			if !v.Stack.TransitionRunning() {
				return true
			}
			return false
		})

		group := global.GetGroup(ctx, gad)
		v.currentGroup = group

		// get existing list
		listI := v.listStack.ChildByName(gad.String())
		var list *gtk.ListBox
		if listI != nil {
			list, _ = listI.(*gtk.ListBox)
		} else {
			// create list if we haven't done that before
			// TODO: we need a context here or something so the subscription is canceled if this group is removed
			list = gtk.NewListBox()
			list.AddCSSClass("message-list")
			list.SetSelectionMode(gtk.SelectionNone)
			list.SetAdjustment(v.Scroll.VAdjustment())

			// insert previously loaded messages
			gtkutil.Async(v.ctx, func() func() {
				<-group.EOSE

				for _, evt := range group.Messages {
					v.upsertMessage(list, evt, -1)
				}

				return func() {
					v.Scroll.ScrollToBottom()
				}
			})

			// listen for new messages
			go func() {
				for evt := range group.NewMessage {
					glib.IdleAdd(func() {
						v.upsertMessage(list, evt, -1)
					})
				}
			}()

			// insert in the stack
			v.listStack.AddNamed(list, gad.String())
		}

		// make it visible
		v.listStack.SetVisibleChild(list)

		// recreate composer and forward typing
		v.Composer = composer.NewView(ctx, v, gad)
		composerOverlay.SetChild(v.Composer)
		gtkutil.ForwardTyping(list, v.Composer.Input)
	}

	v.LoadablePage.SetLoading()

	go func() {
		<-global.GetMe(ctx).ListLoaded
		v.ToastOverlay.SetChild(plc)
		v.LoadablePage.SetChild(v.ToastOverlay)
	}()

	messagesViewCSS(v)
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
	// 	summariesButton := hoverpopover.NewPopoverButton(v.initSummariesPopover)
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

	// 	infoButton := hoverpopover.NewPopoverButton(func(popover *gtk.Popover) bool {
	// 		popover.AddCSSClass("message-channel-info-popover")
	// 		popover.SetPosition(gtk.PosBottom)

	// 		label := gtk.NewLabel("")
	// 		label.AddCSSClass("popover-label")
	// 		popover.SetChild(label)

	// 		state := gtkcord.FromContext(v.ctx)
	// 		ch, _ := state.Offline().Channel(v.chID)
	// 		if ch == nil {
	// 			label.SetText(locale.Get("Channel information unavailable."))
	// 			return true
	// 		}

	// 		markup := fmt.Sprintf(
	// 			`<b>%s</b>`,
	// 			html.EscapeString(ch.Name))

	// 		if ch.NSFW {
	// 			markup += fmt.Sprintf(
	// 				"\n<i><small>%s</small></i>",
	// 				locale.Get("This channel is NSFW."))
	// 		}

	// 		if ch.Topic != "" {
	// 			markup += fmt.Sprintf(
	// 				"\n<small>%s</small>",
	// 				html.EscapeString(ch.Topic))
	// 		} else {
	// 			markup += fmt.Sprintf(
	// 				"\n<i><small>%s</small></i>",
	// 				locale.Get("No topic set."))
	// 		}

	// 		label.SetSizeRequest(100, -1)
	// 		label.SetMaxWidthChars(100)
	// 		label.SetWrap(true)
	// 		label.SetWrapMode(pango.WrapWordChar)
	// 		label.SetJustify(gtk.JustifyLeft)
	// 		label.SetXAlign(0)
	// 		label.SetMarkup(markup)
	// 		return true
	// 	})
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
	// 	first.ListBoxRow.AddCSSClass("message-first-prepended")

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

func (v *MessagesView) upsertMessage(list *gtk.ListBox, event *nostr.Event, pos int) {
	id := event.ID
	if _, ok := v.msgs[id]; ok {
		return
	}

	cmessage := NewCozyMessage(v.ctx, event, v)
	row := gtk.NewListBoxRow()
	row.AddCSSClass("message-row")
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

func (v *MessagesView) deleteMessage(list *gtk.ListBox, id string) {
	msg, ok := v.msgs[id]
	if !ok {
		return
	}

	list.Remove(msg)
	delete(v.msgs, id)
}

func (v *MessagesView) lastMessage(list *gtk.ListBox) (messageRow, bool) {
	row, _ := list.LastChild().(*gtk.ListBoxRow)
	if row != nil {
		msg, ok := v.msgs[row.Name()]
		return msg, ok
	}

	return messageRow{}, false
}

func (v *MessagesView) lastUserMessage() *cozyMessage {
	me := global.GetMe(v.ctx)
	var res *cozyMessage

	list := v.visibleList()
	if list == nil {
		return nil
	}

	eachChild(list, func(lbr *gtk.ListBoxRow) bool {
		if msg, ok := v.msgs[lbr.Name()]; ok && msg.event.PubKey == me.PubKey {
			res = msg.message
			return true
		}
		return false
	})

	return res
}

func (v *MessagesView) updateMember(pubkey string) {
	for _, msg := range v.msgs {
		if msg.event.PubKey != pubkey {
			continue
		}
		msg.message.UpdateMember(pubkey)
	}
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

// SendMessage implements composer.Controller.
func (v *MessagesView) SendMessage(msg composer.SendingMessage) {
	// state := gtkcord.FromContext(v.ctx)

	// me, _ := state.Cabinet.Me()
	// if me == nil {
	// 	// Risk of leaking Files is too high. Just explode. This realistically
	// 	// never happens anyway.
	// 	panic("missing state.Cabinet.Me")
	// }

	// info := messageInfo{
	// 	author:    newMessageAuthor(me),
	// 	timestamp: discord.Timestamp(time.Now()),
	// }

	// var flags upsertFlags
	// if v.shouldBeCollapsed(info) {
	// 	flags |= upsertFlagCollapsed
	// }

	// key := messageKeyLocal()
	// row := v.upsertMessage(key, info, flags)

	// m := nostr.Event{
	// 	ChannelID: v.chID,
	// 	GuildID:   v.guildID,
	// 	Content:   msg.Content,
	// 	Timestamp: discord.NowTimestamp(),
	// 	Author:    *me,
	// }

	// if msg.ReplyingTo.IsValid() {
	// 	m.Reference = &nostr.EventReference{
	// 		ChannelID: v.chID,
	// 		GuildID:   v.guildID,
	// 		MessageID: msg.ReplyingTo,
	// 	}
	// }

	// gtk.BaseWidget(row).AddCSSClass("message-sending")
	// row.Update(&gateway.MessageCreateEvent{Message: m})

	// uploading := newUploadingLabel(v.ctx, len(msg.Files))
	// uploading.SetVisible(len(msg.Files) > 0)

	// content := row.Content()
	// content.Update(&m, uploading)

	// // Use the Background context so things keep getting updated when we switch
	// // away.
	// gtkutil.Async(context.Background(), func() func() {
	// 	sendData := api.SendMessageData{
	// 		Content:   m.Content,
	// 		Reference: m.Reference,
	// 		Nonce:     key.Nonce(),
	// 		AllowedMentions: &api.AllowedMentions{
	// 			RepliedUser: &msg.ReplyMention,
	// 			Parse: []api.AllowedMentionType{
	// 				api.AllowUserMention,
	// 				api.AllowRoleMention,
	// 				api.AllowEveryoneMention,
	// 			},
	// 		},
	// 	}

	// 	// Ensure that we open ALL files and defer-close them. Otherwise, we'll
	// 	// leak files.
	// 	for _, file := range msg.Files {
	// 		f, err := file.Open()
	// 		if err != nil {
	// 			glib.IdleAdd(func() { uploading.AppendError(err) })
	// 			continue
	// 		}

	// 		// This defer executes once we return (like all defers do).
	// 		defer f.Close()

	// 		sendData.Files = append(sendData.Files, sendpart.File{
	// 			Name:   file.Name,
	// 			Reader: wrappedReader{f, uploading},
	// 		})
	// 	}

	// 	state := state.Online()
	// 	_, err := state.SendMessageComplex(m.ChannelID, sendData)

	// 	return func() {
	// 		gtk.BaseWidget(row).RemoveCSSClass("message-sending")

	// 		if err != nil {
	// 			uploading.AppendError(err)
	// 		}

	// 		// We'll let the gateway echo back our own event that's identified
	// 		// using the nonce.
	// 		uploading.SetVisible(uploading.HasErrored())
	// 	}
	// })
}

// ScrollToMessage scrolls to the message with the given ID.
func (v *MessagesView) ScrollToMessage(id string) {
	msg, ok := v.msgs[id]
	if !ok {
		slog.Warn(
			"tried to scroll to non-existent message",
			"id", id)
		return
	}

	v.listStack.SetVisibleChildName(msg.message.Content.view.gad.String())

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

// AddToast adds a toast to the message view.
func (v *MessagesView) AddToast(toast *adw.Toast) {
	v.ToastOverlay.AddToast(toast)
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

	msg.AddCSSClass("message-replying")
	v.Composer.StartReplyingTo(msg.message.Event)
}

// StopEditing implements composer.Controller.
func (v *MessagesView) StopEditing() {
	v.stopEditingOrReplying()
}

// StopReplying implements composer.Controller.
func (v *MessagesView) StopReplying() {
	v.stopEditingOrReplying()
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
