package guilds

import (
	"context"

	"fiatjaf.com/shiitake/components/hoverpopover"
	"fiatjaf.com/shiitake/sidebar/sidebutton"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
)

// Guild is a widget showing a single guild icon.
type Guild struct {
	*sidebutton.Button
	ctx     context.Context
	popover *hoverpopover.MarkupHoverPopover
	id      string
	name    string
}

var guildCSS = cssutil.Applier("guild-guild", `
	.guild-name {
		font-weight: bold;
	}
`)

func NewGuild(ctx context.Context, id string) *Guild {
	g := &Guild{ctx: ctx, id: id}
	g.Button = sidebutton.NewButton(ctx, func() {
		parent := gtk.BaseWidget(g.Button.Parent())
		parent.ActivateAction("win.open-guild", glib.NewVariantString(id))
	})

	g.popover = hoverpopover.NewMarkupHoverPopover(g.Button, func(w *hoverpopover.MarkupHoverPopoverWidget) bool {
		w.AddCSSClass("guild-name-popover")
		w.SetPosition(gtk.PosRight)
		w.Label.AddCSSClass("guild-name")
		w.Label.SetText(g.name)
		return true
	})

	g.SetUnavailable()
	guildCSS(g)
	return g
}

// ID returns the guild ID.
func (g *Guild) ID() string { return g.id }

// Name returns the guild's name.
func (g *Guild) Name() string { return g.name }

// Invalidate invalidates and updates the state of the guild.
func (g *Guild) Invalidate() {
	// guild, err := state.Cabinet.Guild(g.id)
	// if err != nil {
	// 	g.SetUnavailable()
	// 	return
	// }

	// g.Update(guild)
}

// SetUnavailable sets the guild as unavailable. It stays unavailable until
// either Invalidate sees it or Update is called on it.
func (g *Guild) SetUnavailable() {
	g.name = "(guild unavailable)"
	g.SetSensitive(false)

	if g.Icon.Initials() == "" {
		g.Icon.SetInitials("?")
	}
}

// Update updates the guild with the given Discord object.
func (g *Guild) Update(guild *discord.Guild) {
	// g.name = guild.Name

	// g.SetSensitive(true)
	// g.Icon.SetInitials(guild.Name)
	// g.Icon.SetFromURL(gtkcord.InjectAvatarSize(guild.IconURL()))
}

func (g *Guild) viewChild() {}
