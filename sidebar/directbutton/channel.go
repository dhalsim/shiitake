package directbutton

import (
	"context"

	"fiatjaf.com/shiitake/sidebar/sidebutton"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
)

type ChannelButton struct {
	*sidebutton.Button
	id string
}

var channelCSS = cssutil.Applier("dmbutton-channel", `
`)

func NewChannelButton(ctx context.Context, id string) *ChannelButton {
	ch := ChannelButton{id: id}
	ch.Button = sidebutton.NewButton(ctx, func() {
		parent := gtk.BaseWidget(ch.Button.Parent())
		parent.ActivateAction("win.open-channel", glib.NewVariantString(id))
	})
	channelCSS(ch)
	return &ch
}

// ID returns the channel ID.
func (c *ChannelButton) ID() string { return c.id }

// Invalidate invalidates and updates the state of the channel.
func (c *ChannelButton) Invalidate() {
	// state := gtkcord.FromContext(c.Context())

	// ch, err := state.Cabinet.Channel(c.id)
	// if err != nil {
	// 	log.Println("dmbutton.Channel.Invalidate: cannot fetch channel:", err)
	// 	return
	// }

	// c.Update(ch)
	// c.InvalidateUnread()
}

// Update updates the channel with the given Discord object.
func (c *ChannelButton) Update(ch *discord.Channel) {
	// name := gtkcord.ChannelName(ch)

	// var iconURL string
	// if ch.Icon != "" {
	// 	iconURL = ch.IconURL()
	// } else if len(ch.DMRecipients) == 1 {
	// 	iconURL = ch.DMRecipients[0].AvatarURL()
	// }

	// c.Button.SetTooltipText(name)
	// c.Icon.SetInitials(name)
	// c.Icon.SetFromURL(iconURL)
}

// InvalidateUnread invalidates the guild's unread state.
func (c *ChannelButton) InvalidateUnread() {
	// state := gtkcord.FromContext(c.Context())
	// unreads := state.ChannelCountUnreads(c.id, ningen.UnreadOpts{})

	// indicator := state.ChannelIsUnread(c.id, ningen.UnreadOpts{})
	// if indicator != ningen.ChannelRead && unreads == 0 {
	// 	unreads = 1
	// }

	// c.SetIndicator(indicator)
	// c.Mentions.SetCount(unreads)
}
