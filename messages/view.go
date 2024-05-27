package messages

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"slices"

	"fiatjaf.com/shiitake/global"
	"fiatjaf.com/shiitake/messages/composer"
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
)

type messageRow struct {
	*gtk.ListBoxRow
	message *cozyMessage
	event   *nostr.Event
}

type messageInfo struct {
	id        string
	author    string
	timestamp nostr.Timestamp
}

func newMessageInfo(msg *nostr.Event) messageInfo {
	return messageInfo{
		id:        msg.ID,
		author:    msg.PubKey,
		timestamp: msg.CreatedAt,
	}
}

// MessagesView is a message view widget.
type MessagesView struct {
	*adaptive.LoadablePage
	focused gtk.Widgetter

	ToastOverlay    *adw.ToastOverlay
	LoadMore        *gtk.Button
	Scroll          *autoscroll.Window
	List            *gtk.ListBox
	Composer        *composer.View
	TypingIndicator *TypingIndicator

	msgs  map[string]messageRow
	Group global.Group

	state struct {
		row      *gtk.ListBoxRow
		replying bool
	}

	ctx context.Context
	gad nip29.GroupAddress
}

var viewCSS = cssutil.Applier("message-view", `
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
	clamp.SetMaximumSize(messagesWidth.Value())
	// Set tightening threshold to 90% of the clamp's width.
	clamp.SetTighteningThreshold(int(float64(messagesWidth.Value()) * 0.9))
}

// NewView creates a new MessagesView widget associated with the given channel ID. All
// methods call on it will act on that channel.
func NewMessagesView(ctx context.Context, gad nip29.GroupAddress) *MessagesView {
	v := &MessagesView{
		msgs: make(map[string]messageRow),
		gad:  gad,
		ctx:  ctx,
	}

	v.LoadMore = gtk.NewButton()
	v.LoadMore.AddCSSClass("message-show-more")
	v.LoadMore.SetLabel(locale.Get("Show More"))
	v.LoadMore.SetHExpand(true)
	v.LoadMore.SetSensitive(true)
	v.LoadMore.ConnectClicked(v.loadMore)

	v.List = gtk.NewListBox()
	v.List.AddCSSClass("message-list")
	v.List.SetSelectionMode(gtk.SelectionNone)

	clampBox := gtk.NewBox(gtk.OrientationVertical, 0)
	clampBox.SetHExpand(true)
	clampBox.SetVExpand(true)
	clampBox.Append(v.LoadMore)
	clampBox.Append(v.List)

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
		const undershootClass = "undershoot-bottom"

		value := scrollAdjustment.Value()
		upper := scrollAdjustment.Upper()
		psize := scrollAdjustment.PageSize()
		if value < upper-psize {
			v.Scroll.AddCSSClass(undershootClass)
		} else {
			v.Scroll.RemoveCSSClass(undershootClass)
		}
	})

	vp := v.Scroll.Viewport()
	vp.SetScrollToFocus(true)
	v.List.SetAdjustment(v.Scroll.VAdjustment())

	v.Composer = composer.NewView(ctx, v, gad)
	gtkutil.ForwardTyping(v.List, v.Composer.Input)

	v.TypingIndicator = NewTypingIndicator(ctx, gad)
	v.TypingIndicator.SetHExpand(true)
	v.TypingIndicator.SetVAlign(gtk.AlignStart)

	composerOverlay := gtk.NewOverlay()
	composerOverlay.AddOverlay(v.TypingIndicator)
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

	// This becomes the outermost widget.
	v.focused = v.ToastOverlay

	v.LoadablePage = adaptive.NewLoadablePage()
	v.LoadablePage.SetTransitionDuration(125)
	v.setPageToMain()

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

	go func() {
		group := global.GetGroup(ctx, gad)
		for _, evt := range group.Messages {
			v.upsertMessage(evt, -1)
		}
		for evt := range group.NewMessage {
			v.upsertMessage(evt, -1)
		}
	}()

	// state := gtkcord.FromContext(v.ctx)
	// if ch, err := state.Cabinet.Channel(v.chID); err == nil {
	// 	v.chName = ch.Name
	// 	v.guildID = ch.GuildID
	// }

	// state.BindWidget(v, func(ev gateway.Event) {
	// 	switch ev := ev.(type) {
	// 	case *gateway.MessageCreateEvent:
	// 		if ev.ChannelID != v.chID {
	// 			return
	// 		}

	// 		// Use this to update existing messages' members as well.
	// 		if ev.Member != nil {
	// 			v.updateMember(ev.Member)
	// 		}

	// 		if ev.Nonce != "" {
	// 			// Try and look up the nonce.
	// 			key := messageKeyNonce(ev.Nonce)

	// 			if msg, ok := v.msgs[key]; ok {
	// 				delete(v.msgs, key)

	// 				key = messageKeyID(ev.ID)
	// 				// Known sent message. Update this instead.
	// 				v.msgs[key] = msg

	// 				msg.ListBoxRow.SetName(string(key))
	// 				msg.message.Update(ev)
	// 				return
	// 			}
	// 		}

	// 		if !v.ignoreMessage(&ev.Message) {
	// 			msg := v.upsertMessage(ev.ID, newMessageInfo(&ev.Message), 0)
	// 			msg.Update(ev)
	// 		}

	// 	case *gateway.MessageUpdateEvent:
	// 		if ev.ChannelID != v.chID {
	// 			return
	// 		}

	// 		m, err := state.Cabinet.Message(ev.ChannelID, ev.ID)
	// 		if err == nil && !v.ignoreMessage(&ev.Message) {
	// 			msg := v.upsertMessage(ev.ID, newMessageInfo(m), 0)
	// 			msg.Update(&gateway.MessageCreateEvent{
	// 				Message: *m,
	// 				Member:  ev.Member,
	// 			})
	// 		}

	// 	case *gateway.MessageDeleteEvent:
	// 		if ev.ChannelID != v.chID {
	// 			return
	// 		}

	// 		v.deleteMessage(ev.ID)

	// 	case *gateway.MessageReactionAddEvent:
	// 		if ev.ChannelID != v.chID {
	// 			return
	// 		}
	// 		v.updateMessageReactions(ev.MessageID)

	// 	case *gateway.MessageReactionRemoveEvent:
	// 		if ev.ChannelID != v.chID {
	// 			return
	// 		}
	// 		v.updateMessageReactions(ev.MessageID)

	// 	case *gateway.MessageReactionRemoveAllEvent:
	// 		if ev.ChannelID != v.chID {
	// 			return
	// 		}
	// 		v.updateMessageReactions(ev.MessageID)

	// 	case *gateway.MessageReactionRemoveEmojiEvent:
	// 		if ev.ChannelID != v.chID {
	// 			return
	// 		}
	// 		v.updateMessageReactions(ev.MessageID)

	// 	case *gateway.MessageDeleteBulkEvent:
	// 		if ev.ChannelID != v.chID {
	// 			return
	// 		}

	// 		for _, id := range ev.IDs {
	// 			v.deleteMessage(id)
	// 		}

	// 	case *gateway.GuildMemberAddEvent:
	// 		log.Println("TODO: handle GuildMemberAddEvent")

	// 	case *gateway.GuildMemberUpdateEvent:
	// 		if ev.GuildID != v.guildID {
	// 			return
	// 		}

	// 		member, _ := state.Cabinet.Member(ev.GuildID, ev.User.ID)
	// 		if member != nil {
	// 			v.updateMember(member)
	// 		}

	// 	case *gateway.GuildMemberRemoveEvent:
	// 		log.Println("TODO: handle GuildMemberDeleteEvent")

	// 	case *gateway.GuildMembersChunkEvent:
	// 		// TODO: Discord isn't sending us this event. I'm not sure why.
	// 		// Their client has to work somehow. Maybe they use the right-side
	// 		// member list?
	// 		if ev.GuildID != v.guildID {
	// 			return
	// 		}

	// 		for i := range ev.Members {
	// 			v.updateMember(&ev.Members[i])
	// 		}

	// 	case *gateway.ConversationSummaryUpdateEvent:
	// 		if ev.ChannelID != v.chID {
	// 			return
	// 		}

	// 		v.updateSummaries(ev.Summaries)
	// 	}
	// })

	// gtkutil.BindActionCallbackMap(v.List, map[string]gtkutil.ActionCallback{
	// 	"messages.scroll-to": {
	// 		ArgType: gtkcord.SnowflakeVariant,
	// 		Func: func(args *glib.Variant) {
	// 			id := string(args.Int64())

	// 			msg, ok := v.msgs[messageKeyID(id)]
	// 			if !ok {
	// 				slog.Warn(
	// 					"tried to scroll to non-existent message",
	// 					"id", id)
	// 				return
	// 			}

	// 			if !msg.ListBoxRow.GrabFocus() {
	// 				slog.Warn(
	// 					"failed to grab focus of message",
	// 					"id", id)
	// 			}
	// 		},
	// 	},
	// })

	v.load()

	viewCSS(v)
	return v
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

func (v *MessagesView) load() {
	log.Println("loading message view for", v.gad)

	v.LoadablePage.SetLoading()
	v.unload()

	// state := gtkcord.FromContext(v.ctx).Online()

	gtkutil.Async(v.ctx, func() func() {
		msgs := make([]*nostr.Event, 0)
		// msgs, err := state.Messages(v.chID, 15)
		// if err != nil {
		// 	return func() {
		// 		v.LoadablePage.SetError(err)
		// 	}
		// }

		slices.SortFunc(msgs, func(a, b *nostr.Event) int { return int(a.CreatedAt) - int(b.CreatedAt) })

		return func() {
			v.setPageToMain()
			v.Scroll.ScrollToBottom()
		}
	})
}

func (v *MessagesView) loadMore() {
	firstRow, ok := v.firstMessage()
	if !ok {
		return
	}

	firstID := firstRow.event.ID

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

func (v *MessagesView) setPageToMain() {
	v.LoadablePage.SetChild(v.focused)
}

func (v *MessagesView) unload() {
	for k, msg := range v.msgs {
		v.List.Remove(msg)
		delete(v.msgs, k)
	}
}

func (v *MessagesView) upsertMessage(event *nostr.Event, pos int) *cozyMessage {
	id := event.ID

	if msg, ok := v.msgs[id]; ok {
		return msg.message
	}

	msg := v.createMessage(id, event)
	v.msgs[id] = msg

	v.List.Insert(msg.ListBoxRow, pos)
	v.List.SetFocusChild(msg.ListBoxRow)
	return msg.message
}

func (v *MessagesView) createMessage(id string, evt *nostr.Event) messageRow {
	message := NewCozyMessage(v.ctx, evt, v)

	row := gtk.NewListBoxRow()
	row.AddCSSClass("message-row")
	row.SetName(id)
	row.SetChild(message)

	return messageRow{
		ListBoxRow: row,
		message:    message,
		event:      evt,
	}
}

func (v *MessagesView) deleteMessage(id string) {
	msg, ok := v.msgs[id]
	if !ok {
		return
	}

	if redactMessages.Value() && msg.message != nil {
		msg.message.Content.Redact()
		return
	}

	v.List.Remove(msg)
	delete(v.msgs, id)
}

func (v *MessagesView) lastMessage() (messageRow, bool) {
	row, _ := v.List.LastChild().(*gtk.ListBoxRow)
	if row != nil {
		msg, ok := v.msgs[row.Name()]
		return msg, ok
	}

	return messageRow{}, false
}

func (v *MessagesView) lastUserMessage() *cozyMessage {
	me := global.GetMe(v.ctx)

	var msg *cozyMessage
	v.eachMessage(func(row messageRow) bool {
		if row.event.PubKey != me.PubKey {
			// keep looping
			return false
		}
		msg = row.message
		return true
	})

	return msg
}

func (v *MessagesView) firstMessage() (messageRow, bool) {
	row, _ := v.List.FirstChild().(*gtk.ListBoxRow)
	if row != nil {
		msg, ok := v.msgs[row.Name()]
		return msg, ok
	}

	return messageRow{}, false
}

// eachMessage iterates over each message in the view, starting from the bottom.
// If the callback returns true, the loop will break.
func (v *MessagesView) eachMessage(f func(messageRow) bool) {
	row, _ := v.List.LastChild().(*gtk.ListBoxRow)
	for row != nil {
		key := row.Name()

		m, ok := v.msgs[key]
		if ok {
			if f(m) {
				break
			}
		}

		// This repeats until index is -1, at which the loop will break.
		row, _ = row.PrevSibling().(*gtk.ListBoxRow)
	}
}

func (v *MessagesView) eachMessageFromUser(pubkey string, f func(messageRow) bool) {
	v.eachMessage(func(row messageRow) bool {
		if row.event.PubKey == pubkey {
			return f(row)
		}
		return false
	})
}

func (v *MessagesView) updateMember(pubkey string) {
	v.eachMessage(func(row messageRow) bool {
		if row.event.PubKey != pubkey {
			// keep looping
			return false
		}

		row.message.UpdateMember(pubkey)
		return false // keep looping
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
	if !v.List.ActivateAction("messages.scroll-to", glib.NewVariantString(id)) {
		slog.Error(
			"cannot emit messages.scroll-to signal",
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

// Delete deletes the message with the given ID. It may prompt the user to
// confirm the deletion.
func (v *MessagesView) Delete(id string) {
	if !askBeforeDelete.Value() {
		v.delete(id)
		return
	}

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

	// state := gtkcord.FromContext(v.ctx)
	// go func() {
	// 	// This is a fairly important operation, so ensure it goes through even
	// 	// if the user switches away.
	// 	state = state.WithContext(context.Background())

	// 	if err := state.DeleteMessage(v.chID, id, ""); err != nil {
	// 		app.Error(v.ctx, errors.Wrap(err, "cannot delete message"))
	// 	}
	// }()
}

func (v *MessagesView) onScrollBottomed() {
	if v.IsActive() {
		v.MarkRead()
	}

	// Try to clean up the top messages.
	// Fast path: check our cache first.
	if len(v.msgs) > idealMaxCount {
		var count int

		row, _ := v.List.LastChild().(*gtk.ListBoxRow)
		for row != nil {
			next, _ := row.PrevSibling().(*gtk.ListBoxRow)

			if count < idealMaxCount {
				count++
			} else {
				// Start purging messages.
				v.List.Remove(row)
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
