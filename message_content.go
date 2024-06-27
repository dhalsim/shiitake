package main

import (
	"context"

	"fiatjaf.com/shiitake/global"
	"github.com/diamondburned/chatkit/components/author"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gotkit/app/locale"
	"github.com/diamondburned/gotkit/gtkutil"
	"github.com/diamondburned/gotkit/gtkutil/imgutil"
	"github.com/nbd-wtf/go-nostr"
)

// Content is the message content widget.
type Content struct {
	*gtk.Box
	ctx   context.Context
	menu  *gio.Menu
	child []gtk.Widgetter

	MessageID string
}

func NewContent(ctx context.Context, event *nostr.Event) *Content {
	c := Content{
		ctx:       ctx,
		child:     make([]gtk.Widgetter, 0, 2),
		MessageID: event.ID,
	}
	c.Box = gtk.NewBox(gtk.OrientationVertical, 0)

	c.clear()

	msg := gtk.NewLabel("")
	msg.SetText(event.Content)
	msg.SetSelectable(true)
	msg.SetHExpand(true)
	msg.SetXAlign(0)
	msg.SetWrap(true)
	msg.SetWrapMode(pango.WrapWordChar)
	msg.ConnectActivateLink(func(uri string) bool {
		return true
	})
	fixNatWrap(msg)
	c.append(msg)

	// if m.Reference != nil {
	// 	w := c.newReplyBox(m)
	// 	c.append(w)
	// }

	// var messageMarkup string
	// switch m.Type {
	// case discord.GuildMemberJoinMessage:
	// 	messageMarkup = locale.Get("Joined the server.")
	// case discord.CallMessage:
	// 	messageMarkup = locale.Get("Calling you.")
	// case discord.ChannelIconChangeMessage:
	// 	messageMarkup = locale.Get("Changed the channel icon.")
	// case discord.ChannelNameChangeMessage:
	// 	messageMarkup = locale.Get("Changed the channel name to #%s.", html.EscapeString(m.Content))
	// case discord.ChannelPinnedMessage:
	// 	messageMarkup = locale.Get(`Pinned <a href="#message/%d">a message</a>.`, m.ID)
	// case discord.RecipientAddMessage, discord.RecipientRemoveMessage:
	// 	mentioned := state.MemberMarkup(m.GuildID, &m.Mentions[0], author.WithMinimal())
	// 	switch m.Type {
	// 	case discord.RecipientAddMessage:
	// 		messageMarkup = locale.Get("Added %s to the group.", mentioned)
	// 	case discord.RecipientRemoveMessage:
	// 		messageMarkup = locale.Get("Removed %s from the group.", mentioned)
	// 	}
	// }

	// switch {
	// case messageMarkup != "":
	// 	msg := gtk.NewLabel("")
	// 	msg.SetMarkup(messageMarkup)
	// 	msg.SetHExpand(true)
	// 	msg.SetXAlign(0)
	// 	msg.SetWrap(true)
	// 	msg.SetWrapMode(pango.WrapWordChar)
	// 	msg.ConnectActivateLink(func(uri string) bool {
	// 		if !strings.HasPrefix(uri, "#") {
	// 			return false // not our link
	// 		}

	// 		parts := strings.SplitN(uri, "/", 2)
	// 		if len(parts) != 2 {
	// 			return true // pretend we've handled this because of #
	// 		}

	// 		switch strings.TrimPrefix(parts[0], "#") {
	// 		case "message":
	// 			if id, _ := discord.ParseSnowflake(parts[1]); id.IsValid() {
	// 				c.view.ScrollToMessage(string(id))
	// 			}
	// 		}

	// 		return true
	// 	})
	// 	fixNatWrap(msg)
	// 	c.append(msg)
	// }

	// c.SetReactions(m.Reactions)
	c.setMenu()

	return &c
}

// SetExtraMenu implements ExtraMenuSetter.
func (c *Content) SetExtraMenu(menu gio.MenuModeller) {
	c.menu = gio.NewMenu()
	c.menu.InsertSection(0, locale.Get("Message"), menu)

	c.setMenu()
}

type extraMenuSetter interface{ SetExtraMenu(gio.MenuModeller) }

var (
	_ extraMenuSetter = (*gtk.TextView)(nil)
	_ extraMenuSetter = (*gtk.Label)(nil)
)

func (c *Content) setMenu() {
	var menu gio.MenuModeller
	if c.menu != nil {
		menu = c.menu // because a nil interface{} != nil *T
	}

	for _, child := range c.child {
		// Manually check on child to allow certain widgets to override the
		// method.
		s, ok := child.(extraMenuSetter)
		if ok {
			s.SetExtraMenu(menu)
		}

		gtkutil.WalkWidget(c.Box, func(w gtk.Widgetter) bool {
			s, ok := w.(extraMenuSetter)
			if ok {
				s.SetExtraMenu(menu)
			}
			return false
		})
	}
}

func (c *Content) newReplyBox(m *nostr.Event) gtk.Widgetter {
	box := gtk.NewBox(gtk.OrientationVertical, 0)

	// state := gtkcord.FromContext(c.ctx)

	// referencedMsg := m.ReferencedMessage
	// if referencedMsg == nil {
	// 	referencedMsg, _ = state.Cabinet.Message(m.Reference.ChannelID, m.Reference.MessageID)
	// }

	// if referencedMsg == nil {
	// 	slog.Warn(
	// 		"Cannot display message reference because the message is not found",
	// 		"channel_id", m.ChannelID,
	// 		"guild_id", m.GuildID,
	// 		"id", m.ID,
	// 		"id_reference", m.Reference.MessageID)

	// 	header := gtk.NewLabel("Unknown message.")
	// 	box.Append(header)

	// 	return box
	// }

	// member, _ := state.Cabinet.Member(m.Reference.GuildID, referencedMsg.Author.ID)
	chip := newAuthorChip(c.ctx, "", global.GetUser(c.ctx, m.PubKey))
	chip.SetHAlign(gtk.AlignStart)
	chip.Unpad()
	box.Append(chip)

	// if preview := state.MessagePreview(referencedMsg); preview != "" {
	// 	// Force single line.
	// 	preview = strings.ReplaceAll(preview, "\n", "  ")
	// 	markup := fmt.Sprintf(
	// 		`<a href="dissent://reply">%s</a>`,
	// 		html.EscapeString(preview),
	// 	)

	// 	reply := gtk.NewLabel(markup)
	// 	reply.SetUseMarkup(true)
	// 	reply.SetTooltipText(preview)
	// 	reply.SetEllipsize(pango.EllipsizeEnd)
	// 	reply.SetLines(1)
	// 	reply.SetXAlign(0)
	// 	reply.ConnectActivateLink(func(link string) bool {
	// 		slog.Debug(
	// 			"Activated message reference link",
	// 			"link", link,
	// 			"message_id", m.ID,
	// 			"reference_id", referencedMsg.ID)

	// 		if link != "dissent://reply" {
	// 			return false
	// 		}

	// 		if !c.ActivateAction("messages.scroll-to", gtkcord.NewMessageIDVariant(m.ID)) {
	// TODO: call function on parent directly instead of emitting this event thing
	// 			slog.Error(
	// 				"Failed to activate messages.scroll-to",
	// 				"id", m.ID)
	// 		}

	// 		return true
	// 	})

	// 	box.Append(reply)
	// }

	// if state.UserIsBlocked(referencedMsg.Author.ID) {
	// 	blockedCSS(box)
	// }

	return box
}

func (c *Content) newInteractionBox(m *nostr.Event) gtk.Widgetter {
	box := gtk.NewBox(gtk.OrientationHorizontal, 0)

	// state := gtkcord.FromContext(c.ctx)

	// if !showBlockedMessages.Value() && state.UserIsBlocked(m.Interaction.User.ID) {
	// 	header := gtk.NewLabel("Blocked user.")
	// 	box.Append(header)

	// 	blockedCSS(box)
	// 	return box
	// }

	chip := newAuthorChip(c.ctx, "", global.GetUser(c.ctx, m.PubKey))
	chip.SetHAlign(gtk.AlignStart)
	chip.Unpad()
	box.Append(chip)

	// nameLabel := gtk.NewLabel(m.Interaction.Name)
	// if m.Interaction.Type == discord.CommandInteractionType {
	// 	nameLabel.SetText("/" + m.Interaction.Name)
	// }
	// nameLabel.SetTooltipText(m.Interaction.Name)
	// nameLabel.SetEllipsize(pango.EllipsizeEnd)
	// nameLabel.SetXAlign(0)
	// box.Append(nameLabel)

	// if state.UserIsBlocked(m.Interaction.User.ID) {
	// 	blockedCSS(box)
	// }

	return box
}

func (c *Content) append(w gtk.Widgetter) {
	c.Box.Append(w)
	c.child = append(c.child, w)
}

func (c *Content) clear() {
	for i, child := range c.child {
		c.Box.Remove(child)
		c.child[i] = nil
	}
	c.child = c.child[:0]
}

// Redact clears the content widget.
func (c *Content) Redact() {
	c.clear()

	red := gtk.NewLabel(locale.Get("Redacted."))
	red.SetXAlign(0)
	c.append(red)
}

// rgba(111, 120, 219, 0.3)
const defaultMentionColor = "#6F78DB"

func newAuthorChip(ctx context.Context, guildID string, user global.User) *author.Chip {
	name := user.ShortName()
	color := defaultMentionColor

	// if user.Member != nil {
	// 	if user.Member.Nick != "" {
	// 		name = user.Member.Nick
	// 	}

	// 	s := gtkcord.FromContext(ctx)
	// 	c, ok := state.MemberColor(user.Member, func(id discord.RoleID) *discord.Role {
	// 		r, _ := s.Cabinet.Role(guildID, id)
	// 		return r
	// 	})
	// 	if ok {
	// 		color = c.String()
	// 	}
	// }

	chip := author.NewChip(ctx, imgutil.HTTPProvider)
	chip.SetName(name)
	chip.SetColor(color)
	chip.SetAvatar(user.Picture)

	return chip
}
