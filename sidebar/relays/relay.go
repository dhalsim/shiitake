package relays

import (
	"context"
	"strings"

	"fiatjaf.com/shiitake/components/hoverpopover"
	"fiatjaf.com/shiitake/global"
	"fiatjaf.com/shiitake/sidebar/sidebutton"
	"fiatjaf.com/shiitake/utils"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
)

// Guild is a widget showing a single guild icon.
type Relay struct {
	*sidebutton.Button
	ctx     context.Context
	popover *hoverpopover.MarkupHoverPopover
	url     string
	name    string
	image   string
}

var guildCSS = cssutil.Applier("guild-guild", `
	.relay-name {
		font-weight: bold;
	}
`)

func NewRelay(ctx context.Context, relay *global.Relay) *Relay {
	g := &Relay{
		ctx:   ctx,
		url:   relay.URL,
		name:  relay.Name,
		image: relay.Image,
	}

	g.name = relay.URL

	g.Button = sidebutton.NewButton(ctx, func() {
		parent := gtk.BaseWidget(g.Button.Parent())
		parent.ActivateAction("win.open-relay", utils.NewRelayURLVariant(relay.URL))
	})

	g.popover = hoverpopover.NewMarkupHoverPopover(g.Button, func(w *hoverpopover.MarkupHoverPopoverWidget) bool {
		w.AddCSSClass("relay-name-popover")
		w.SetPosition(gtk.PosRight)
		w.Label.AddCSSClass("relay-name")
		w.Label.SetText(g.name)
		return true
	})

	guildCSS(g)

	g.SetSensitive(true)
	initials := strings.Join(strings.Split(strings.Split(relay.URL, "://")[1], "."), " ")
	g.Icon.SetInitials(initials)
	if relay.Image != "" {
		g.Icon.SetFromURL(relay.Image)
	}

	return g
}

// ID returns the guild ID.
func (g *Relay) ID() string { return g.url }

// Name returns the guild's name.
func (g *Relay) Name() string { return g.name }

// Invalidate invalidates and updates the state of the guild.
func (g *Relay) Invalidate() {
	// guild, err := state.Cabinet.Guild(g.id)
	// if err != nil {
	// 	g.SetUnavailable()
	// 	return
	// }

	// g.Update(guild)
}

func (g *Relay) viewChild() {}
