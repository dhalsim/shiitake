package main

import (
	"context"

	"fiatjaf.com/nostr-gtk/components/avatar"
	"fiatjaf.com/shiitake/components/autoscroll"
	"fiatjaf.com/shiitake/global"
	"fiatjaf.com/shiitake/utils"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/gtkutil"
	"github.com/nbd-wtf/go-nostr"
)

type GroupView struct {
	*adw.ToolbarView
	ctx context.Context

	me      *global.Me
	group   *global.Group
	destroy context.CancelFunc // call this when the group is destroyed so subscriptions will be closed

	chat struct {
		scroll      *autoscroll.Window
		list        *gtk.ListBox
		bottomStack *gtk.Stack
		composer    *ComposerView
		replyingTo  *gtk.ListBoxRow
	}
}

func NewGroupView(ctx context.Context, group *global.Group) *GroupView {
	me := global.GetMe(ctx)
	ctx, cancel := context.WithCancel(ctx)

	v := &GroupView{
		ToolbarView: adw.NewToolbarView(),
		ctx:         ctx,
		me:          me,
		group:       group,
		destroy:     cancel,
	}

	viewStack := adw.NewViewStack()

	// this thing will display the names of each stack item automatically at the top
	// (as long as we add them with AddTitled() and provide a title)
	switcher := adw.NewViewSwitcher()
	switcher.SetPolicy(adw.ViewSwitcherPolicyWide)
	switcher.SetStack(viewStack)

	headerBar := adw.NewHeaderBar()
	headerBar.SetTitleWidget(switcher)
	headerBar.SetShowBackButton(false)
	headerBar.SetShowEndTitleButtons(false)
	headerBar.SetShowStartTitleButtons(false)
	headerBar.SetShowTitle(true)

	// group info
	{
		groupInfo := gtk.NewBox(gtk.OrientationVertical, 0)
		groupInfo.AddCSSClass("p-16")

		picture := avatar.New(ctx, 72, group.Name)
		if group.Picture != "" {
			picture.SetFromURL(group.Picture)
		}
		picture.AddCSSClass("mb-4")
		groupInfo.Append(picture)

		name := gtk.NewLabel(group.Name)
		name.AddCSSClass("title-1")
		name.AddCSSClass("mb-2")
		groupInfo.Append(name)

		id := gtk.NewLabel(group.Address.String())
		id.AddCSSClass("title-3")
		id.AddCSSClass("mb-2")
		groupInfo.Append(id)

		about := gtk.NewLabel(group.About)
		about.AddCSSClass("mb-4")
		groupInfo.Append(about)

		button := gtk.NewButtonWithLabel("Join/Leave")
		button.AddCSSClass("text-2xl")
		button.AddCSSClass("mx-24")
		button.ConnectClicked(func() {
			switch button.Label() {
			case "Join":
				button.SetLabel("Joining...")
				button.SetSensitive(false)
				go func() {
					if err := global.JoinGroup(ctx, group.Address); err != nil {
						win.ErrorToast(err.Error())
					}
					button.SetSensitive(true)
				}()
			case "Leave":
				button.SetLabel("Leaving...")
				button.SetSensitive(false)
				go func() {
					global.LeaveGroup(ctx, group.Address)
					button.SetSensitive(true)
				}()
			}
		})
		groupInfo.Append(button)

		viewStack.AddTitled(groupInfo, "group", "Group")

		// update details when we get an update
		group.OnUpdated(func() {
			if group.Picture != "" {
				picture.SetFromURL(group.Picture)
			}
			name.SetLabel(group.Name)
			id.SetLabel(group.Address.String())
			about.SetLabel(group.About)
		})

		// display either "join" or "leave" at the bottom depending on group membership status
		setJoinOrLeave := func() {
			glib.IdleAdd(func() {
				if v.me.InGroup(v.group.Address) {
					button.SetLabel("Leave")
					button.AddCSSClass("destructive-action")
					button.RemoveCSSClass("suggested-action")
				} else {
					button.SetLabel("Join")
					button.AddCSSClass("suggested-action")
					button.RemoveCSSClass("destructive-action")
				}
			})
		}
		setJoinOrLeave()
		v.me.OnListUpdated(setJoinOrLeave)
	}

	// chat
	{
		joinButton := gtk.NewButtonWithLabel("Join")
		joinButton.SetHExpand(true)
		joinButton.SetHAlign(gtk.AlignFill)
		joinButton.AddCSSClass("p-8")
		joinButton.AddCSSClass("mx-4")
		joinButton.AddCSSClass("my-2")
		joinButton.AddCSSClass("suggested-action")
		joinButton.SetTooltipText("Join Group")
		joinButton.ConnectClicked(func() {
			revert := utils.ButtonLoading(joinButton, "Joining...")

			go func() {
				if err := global.JoinGroup(ctx, group.Address); err != nil {
					win.ErrorToast(err.Error())
				}

				revert()
			}()
		})

		v.chat.list = gtk.NewListBox()
		v.chat.list.SetSelectionMode(gtk.SelectionNone)

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
		clampBox.Append(v.chat.list)

		v.chat.scroll = autoscroll.NewWindow()
		v.chat.scroll.SetVExpand(true)
		v.chat.scroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
		v.chat.scroll.SetPropagateNaturalWidth(true)
		v.chat.scroll.SetPropagateNaturalHeight(true)
		v.chat.scroll.SetChild(clampBox)

		scrollAdjustment := v.chat.scroll.ScrolledWindow.VAdjustment()
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
				v.chat.scroll.ScrolledWindow.RemoveCSSClass("undershoot-bottom")
			}
		})

		vp := v.chat.scroll.Viewport()
		vp.SetScrollToFocus(true)

		lastAppendedAuthor := ""
		insertMessage := func(event *nostr.Event, pos int) {
			id := event.ID

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
			row.AddCSSClass("background")
			row.SetName(id)
			row.SetChild(cmessage)

			v.chat.list.Insert(row, pos)
			v.chat.list.Display().Flush()
			v.chat.list.SetFocusChild(row)
		}

		v.chat.bottomStack = gtk.NewStack()
		v.chat.bottomStack.AddNamed(joinButton, "join")

		chatView := gtk.NewBox(gtk.OrientationVertical, 0)
		chatView.Append(v.chat.scroll)
		chatView.Append(v.chat.bottomStack)

		viewStack.AddTitled(chatView, "chat", "Chat")

		v.ToolbarView.SetHExpand(true)
		v.ToolbarView.SetVExpand(true)
		v.ToolbarView.AddTopBar(headerBar)
		v.ToolbarView.SetContent(viewStack)

		showingLoadMoreAlready := false

		// listen for new messages
		go func() {
			for evt := range group.NewMessage {
				glib.IdleAdd(func() {
					insertMessage(evt, -1)

					if !showingLoadMoreAlready &&
						v.chat.scroll.AllocatedHeight() < int(vp.VAdjustment().Upper()) {
						showingLoadMoreAlready = true
						loadMore.Show()
					}
				})
			}
		}()

		// display either "join" button or composer at the end depending on group membership status
		v.me.OnListUpdated(func() {
			if v.me.InGroup(v.group.Address) {
				if v.chat.composer == nil {
					// composer must be created here, not on GroupView instantiation, otherwise gtk.NewTextInput crashes
					v.chat.composer = NewComposerView(v.ctx, v)
					gtkutil.ForwardTyping(v.chat.list, v.chat.composer.Input)
					v.chat.bottomStack.AddNamed(v.chat.composer, "composer")
				}
				v.chat.bottomStack.SetVisibleChildName("composer")
			} else {
				v.chat.bottomStack.SetVisibleChildName("join")
			}
		})
	}

	// always default to displaying the chat
	viewStack.SetVisibleChildName("chat")

	return v
}

func (v *GroupView) loadMore() {
}

// AddReaction adds an reaction to the message with the given ID.
func (v *GroupView) AddReaction(id string, emoji discord.APIEmoji) {
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
func (v *GroupView) ReplyTo(id string) {
	v.stopEditingOrReplying()

	// msg, ok := v.msgs[id]
	// if !ok || msg.message == nil || msg.message.Event == nil {
	// 	return
	// }

	// v.state.row = msg.ListBoxRow
	// v.state.replying = true

	// v.Composer.StartReplyingTo(msg.message.Event)
}

func (v *GroupView) stopEditingOrReplying() {
	if v.chat.replyingTo != nil {
		v.chat.composer.StopReplying()
		v.chat.replyingTo.RemoveCSSClass("message-replying")
	}
}

// func (v *GroupView) Delete(id string) {
// 	username := "?" // juuust in case
//
// 	row, ok := v.msgs[id]
// 	if ok {
// 		user := global.GetUser(v.ctx, row.message.Event.PubKey)
// 		username = fmt.Sprintf(`<span weight="normal">%s</span>`, user.ShortName())
// 	}
//
// 	window := app.GTKWindowFromContext(v.ctx)
// 	dialog := adw.NewMessageDialog(window,
// 		locale.Get("Delete Message"),
// 		locale.Sprintf("Are you sure you want to delete %s's message?", username))
// 	dialog.SetBodyUseMarkup(true)
// 	dialog.AddResponse("cancel", locale.Get("_Cancel"))
// 	dialog.AddResponse("delete", locale.Get("_Delete"))
// 	dialog.SetResponseAppearance("delete", adw.ResponseDestructive)
// 	dialog.SetDefaultResponse("cancel")
// 	dialog.SetCloseResponse("cancel")
// 	dialog.ConnectResponse(func(response string) {
// 		switch response {
// 		case "delete":
// 			v.delete(id)
// 		}
// 	})
// 	dialog.Show()
// }
//
// func (v *GroupView) delete(id string) {
// 	if msg, ok := v.msgs[id]; ok {
// 		// Visual indicator.
// 		msg.SetSensitive(false)
// 	}
// }

// MarkRead marks the view's latest messages as read.
func (v *GroupView) MarkRead() {
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
	// 	log.Println("message.GroupsView.MarkRead: marked", msgs[0].ID, "as read, last read", readState.LastMessageID)
	// }
}

func (v *GroupView) deleteMessage(id string) {
	eachChild(v.chat.list, func(lbr *gtk.ListBoxRow) bool {
		if lbr.Name() == id {
			v.chat.list.Remove(lbr)
			return true
		}
		return false
	})
}
