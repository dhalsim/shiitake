package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"time"

	"fiatjaf.com/nostr-gtk/components/avatar"
	"fiatjaf.com/shiitake/global"
	"github.com/diamondburned/chatkit/md/hl"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gotkit/app"
	"github.com/diamondburned/gotkit/app/locale"
	"github.com/diamondburned/gotkit/gtkutil"
	"github.com/diamondburned/gotkit/gtkutil/textutil"
	"github.com/nbd-wtf/go-nostr"
)

// message is a base that implements Message.
type message struct {
	parent *Message

	Content *Content
	Event   *nostr.Event
	Menu    *gio.Menu
}

func (msg *message) bind() *gio.Menu {
	if msg.Menu != nil {
		return msg.Menu
	}

	actions := map[string]func(){
		"message.show-source": func() { msg.ShowSource() },
		// "message.reply":       func() { win.main.Groups.ReplyTo(msg.Event.ID) },
	}

	me := global.GetMe(msg.Content.ctx)

	// if me != nil && m.message.PubKey == me.PubKey {
	// 	actions["message.edit"] = func() { m.view().Edit(m.message.ID) }
	// 	actions["message.delete"] = func() { m.view().Delete(m.message.ID) }
	// }

	if me != nil && msg.Event.PubKey == me.PubKey /* TODO: admins should also be able to delete */ {
		// actions["message.delete"] = func() { win.main.Messages.Delete(msg.Event.ID) }
	}

	// 	if channel != nil && (channel.Type == discord.DirectMessage || channel.Type == discord.GroupDM) {
	// 		actions["message.add-reaction"] = func() { m.ShowEmojiChooser() }
	// 	}

	menuItems := []gtkutil.PopoverMenuItem{
		// menuItemIfOK(actions, "Add _Reaction", "message.add-reaction"),
		menuItemIfOK(actions, "_Reply", "message.reply"),
		// menuItemIfOK(actions, "_Edit", "message.edit"),
		menuItemIfOK(actions, "_Delete", "message.delete"),
		menuItemIfOK(actions, "Show _Source", "message.show-source"),
	}

	gtkutil.BindActionMap(msg.parent, actions)
	gtkutil.BindPopoverMenuCustom(msg.parent, gtk.PosTop, menuItems)

	msg.Menu = gtkutil.CustomMenu(menuItems)
	msg.Content.SetExtraMenu(msg.Menu)

	return msg.Menu
}

func menuItemIfOK(actions map[string]func(), label locale.Localized, action string) gtkutil.PopoverMenuItem {
	_, ok := actions[action]
	return gtkutil.MenuItem(label, action, ok)
}

// ShowEmojiChooser opens a Gtk.EmojiChooser popover.
func (msg *message) ShowEmojiChooser() {
	e := gtk.NewEmojiChooser()
	e.SetParent(msg.Content)
	e.SetHasArrow(false)

	e.ConnectEmojiPicked(func(text string) {
		// emoji := discord.APIEmoji(text)
		// win.main.Messages.AddReaction(msg.Content.MessageID, emoji)
	})

	e.Present()
	e.Popup()
}

// ShowSource opens a JSON showing the message JSON.
func (msg *message) ShowSource() {
	d := adw.NewWindow()
	d.SetTitle(locale.Get("View Source"))
	d.SetTransientFor(app.GTKWindowFromContext(msg.Content.ctx))
	d.SetModal(true)
	d.SetDefaultSize(730, 400)

	h := adw.NewHeaderBar()
	h.SetCenteringPolicy(adw.CenteringPolicyStrict)

	buf := gtk.NewTextBuffer(nil)

	if raw, err := json.MarshalIndent(msg.Event, "", "\t"); err != nil {
		buf.SetText("Error marshaling JSON: " + err.Error())
	} else {
		buf.SetText(string(raw))
		hl.Highlight(msg.Content.ctx, buf.StartIter(), buf.EndIter(), "json")
	}

	t := gtk.NewTextViewWithBuffer(buf)
	t.SetEditable(false)
	t.SetCursorVisible(false)
	t.SetWrapMode(gtk.WrapWordChar)
	textutil.SetTabSize(t)

	s := gtk.NewScrolledWindow()
	s.SetVExpand(true)
	s.SetHExpand(true)
	s.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	s.SetChild(t)

	copyBtn := gtk.NewButtonFromIconName("edit-copy-symbolic")
	copyBtn.SetTooltipText(locale.Get("Copy JSON"))
	copyBtn.ConnectClicked(func() {
		// clipboard := win.main.Messages.Clipboard()
		// sourceText := buf.Text(buf.StartIter(), buf.EndIter(), false)
		// clipboard.SetText(sourceText)
	})
	h.PackStart(copyBtn)

	box := gtk.NewBox(gtk.OrientationVertical, 0)
	box.Append(h)
	box.Append(s)

	d.SetContent(box)

	d.Show()
}

type Message struct {
	*gtk.Box
	Avatar *avatar.Avatar

	message
	tooltip string // markup
}

func NewMessage(
	ctx context.Context,
	event *nostr.Event,
	fromLoggedUser bool,
	authorIsTheSameAsPrevious bool,
) *Message {
	m := &Message{
		message: message{
			Content: NewContent(ctx, event),
			Event:   event,
		},
	}
	m.message.parent = m

	guctx, cancel := context.WithTimeout(ctx, time.Second*2)
	user := global.GetUser(guctx, event.PubKey)
	cancel()

	markup := "<b>" + user.ShortName() + "</b>"
	markup += ` <span alpha="75%" size="small">` +
		locale.TimeAgo(event.CreatedAt.Time()) +
		"</span>"

	tooltip := fmt.Sprintf(
		"<b>%s</b> (%s)\n%s",
		html.EscapeString(user.ShortName()), user.Npub(),
		html.EscapeString(locale.Time(event.CreatedAt.Time(), true)),
	)

	// TODO: query tooltip

	topLabel := gtk.NewLabel("")
	topLabel.SetXAlign(0)
	topLabel.SetEllipsize(pango.EllipsizeEnd)
	topLabel.SetSingleLineMode(true)
	topLabel.SetMarkup(markup)
	topLabel.SetTooltipMarkup(tooltip)

	rightBox := gtk.NewBox(gtk.OrientationVertical, 0)
	rightBox.SetHExpand(true)
	rightBox.Append(topLabel)
	rightBox.Append(m.message.Content)

	m.Box = gtk.NewBox(gtk.OrientationHorizontal, 0)

	if !fromLoggedUser {
		avatar := avatar.New(ctx, 30, event.PubKey)
		avatar.SetVAlign(gtk.AlignCenter)
		avatar.SetTooltipMarkup(tooltip)
		avatar.SetFromURL(user.Picture)
		avatar.AddCSSClass("mx-2")
		m.Box.Append(avatar)

		if authorIsTheSameAsPrevious {
			avatar.AddCSSClass("opacity-0")
		}
	} else {
		m.Box.SetHAlign(gtk.AlignEnd)
		m.Box.AddCSSClass("mr-2")
	}

	m.Box.Append(rightBox)

	m.message.bind()

	return m
}
