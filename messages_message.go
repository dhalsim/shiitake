package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"time"

	"fiatjaf.com/shiitake/global"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/chatkit/md/hl"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gotkit/app"
	"github.com/diamondburned/gotkit/app/locale"
	"github.com/diamondburned/gotkit/components/onlineimage"
	"github.com/diamondburned/gotkit/gtkutil"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
	"github.com/diamondburned/gotkit/gtkutil/imgutil"
	"github.com/diamondburned/gotkit/gtkutil/textutil"
	"github.com/nbd-wtf/go-nostr"
)

var blockedCSS = cssutil.Applier("message-blocked", `
.message-blocked {
  transition-property: all;
  transition-duration: 100ms;
}
.message-blocked:not(:hover) {
  opacity: 0.35;
}
`)

// message is a base that implements Message.
type message struct {
	parent *cozyMessage

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
		"message.reply":       func() { msg.Content.view.ReplyTo(msg.Event.ID) },
	}

	me := global.GetMe(msg.Content.ctx)

	// if me != nil && m.message.PubKey == me.PubKey {
	// 	actions["message.edit"] = func() { m.view().Edit(m.message.ID) }
	// 	actions["message.delete"] = func() { m.view().Delete(m.message.ID) }
	// }

	if me != nil && msg.Event.PubKey == me.PubKey /* TODO: admins should also be able to delete */ {
		actions["message.delete"] = func() { msg.Content.view.Delete(msg.Event.ID) }
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

var sourceCSS = cssutil.Applier("message-source", `
.message-source {
  padding: 6px 4px;
  font-family: monospace;
}
`)

// ShowEmojiChooser opens a Gtk.EmojiChooser popover.
func (msg *message) ShowEmojiChooser() {
	e := gtk.NewEmojiChooser()
	e.SetParent(msg.Content)
	e.SetHasArrow(false)

	e.ConnectEmojiPicked(func(text string) {
		emoji := discord.APIEmoji(text)
		msg.Content.view.AddReaction(msg.Content.MessageID, emoji)
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
	d.SetDefaultSize(500, 300)

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
	sourceCSS(t)
	textutil.SetTabSize(t)

	s := gtk.NewScrolledWindow()
	s.SetVExpand(true)
	s.SetHExpand(true)
	s.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	s.SetChild(t)

	copyBtn := gtk.NewButtonFromIconName("edit-copy-symbolic")
	copyBtn.SetTooltipText(locale.Get("Copy JSON"))
	copyBtn.ConnectClicked(func() {
		clipboard := msg.Content.view.Clipboard()
		sourceText := buf.Text(buf.StartIter(), buf.EndIter(), false)
		clipboard.SetText(sourceText)
	})
	h.PackStart(copyBtn)

	box := gtk.NewBox(gtk.OrientationVertical, 0)
	box.Append(h)
	box.Append(s)

	d.SetContent(box)

	d.Show()
}

// cozyMessage is a large cozy message with an avatar.
type cozyMessage struct {
	*gtk.Box
	Avatar *onlineimage.Avatar

	message
	tooltip string // markup
}

var cozyCSS = cssutil.Applier("message-cozy", `
.message-cozy {
  border-radius: 6px;
  padding-bottom: 4px;
  padding-top: 4px;
  padding-right: 8px;
  padding-left: 8px;
  margin-top: 0;
}
.message-cozy-header {
  margin-top: 0px;
  min-height: 0;
  font-size: 0.95em;
}
.message-cozy-avatar {
  padding-right: 8px;
}
.msg-bg-0 { background-color: hsl(0.0, 66%, 89%); }
.msg-bg-1 { background-color: hsl(22.5, 66%, 89%); }
.msg-bg-2 { background-color: hsl(45.0, 66%, 89%); }
.msg-bg-3 { background-color: hsl(67.5, 66%, 89%); }
.msg-bg-4 { background-color: hsl(90.0, 66%, 89%); }
.msg-bg-5 { background-color: hsl(112.5, 66%, 89%); }
.msg-bg-6 { background-color: hsl(135.0, 66%, 89%); }
.msg-bg-7 { background-color: hsl(157.5, 66%, 89%); }
.msg-bg-8 { background-color: hsl(180.0, 66%, 89%); }
.msg-bg-9 { background-color: hsl(202.5, 66%, 89%); }
.msg-bg-a { background-color: hsl(225.0, 66%, 89%); }
.msg-bg-b { background-color: hsl(247.5, 66%, 89%); }
.msg-bg-c { background-color: hsl(270.0, 66%, 89%); }
.msg-bg-d { background-color: hsl(292.5, 66%, 89%); }
.msg-bg-e { background-color: hsl(315.0, 66%, 89%); }
.msg-bg-f { background-color: hsl(337.5, 66%, 89%); }

.dark .msg-bg-0 { background-color: hsl(0.0, 50%, 21%); }
.dark .msg-bg-1 { background-color: hsl(22.5, 50%, 21%); }
.dark .msg-bg-2 { background-color: hsl(45.0, 50%, 21%); }
.dark .msg-bg-3 { background-color: hsl(67.5, 50%, 21%); }
.dark .msg-bg-4 { background-color: hsl(90.0, 50%, 21%); }
.dark .msg-bg-5 { background-color: hsl(112.5, 50%, 21%); }
.dark .msg-bg-6 { background-color: hsl(135.0, 50%, 21%); }
.dark .msg-bg-7 { background-color: hsl(157.5, 50%, 21%); }
.dark .msg-bg-8 { background-color: hsl(180.0, 50%, 21%); }
.dark .msg-bg-9 { background-color: hsl(202.5, 50%, 21%); }
.dark .msg-bg-a { background-color: hsl(225.0, 50%, 21%); }
.dark .msg-bg-b { background-color: hsl(247.5, 50%, 21%); }
.dark .msg-bg-c { background-color: hsl(270.0, 50%, 21%); }
.dark .msg-bg-d { background-color: hsl(292.5, 50%, 21%); }
.dark .msg-bg-e { background-color: hsl(315.0, 50%, 21%); }
.dark .msg-bg-f { background-color: hsl(337.5, 50%, 21%); }
`)

func NewCozyMessage(ctx context.Context, event *nostr.Event, v *MessagesView) *cozyMessage {
	m := &cozyMessage{
		message: message{
			Content: NewContent(ctx, event, v),
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
	topLabel.AddCSSClass("message-cozy-header")
	topLabel.SetXAlign(0)
	topLabel.SetEllipsize(pango.EllipsizeEnd)
	topLabel.SetSingleLineMode(true)
	topLabel.SetMarkup(markup)
	topLabel.SetTooltipMarkup(tooltip)

	rightBox := gtk.NewBox(gtk.OrientationVertical, 0)
	rightBox.SetHExpand(true)
	rightBox.Append(topLabel)
	rightBox.Append(m.message.Content)

	avatar := onlineimage.NewAvatar(ctx, imgutil.HTTPProvider, 12)
	avatar.AddCSSClass("message-cozy-avatar")
	avatar.SetVAlign(gtk.AlignCenter)
	avatar.EnableAnimation().OnHover()
	avatar.SetTooltipMarkup(tooltip)
	avatar.SetFromURL(user.Picture)

	m.Box = gtk.NewBox(gtk.OrientationHorizontal, 0)
	m.Box.Append(avatar)
	m.Box.Append(rightBox)
	m.Box.AddCSSClass(fmt.Sprintf("msg-bg-%s", event.PubKey[63:64]))
	align := gtk.AlignStart
	if event.PubKey == v.loggedUser {
		align = gtk.AlignEnd
	}
	m.Box.SetHAlign(align)

	m.message.bind()

	cozyCSS(m)
	return m
}
