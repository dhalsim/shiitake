package relays

import (
	"context"
	"strings"

	"fiatjaf.com/shiitake/components/hoverpopover"
	"fiatjaf.com/shiitake/sidebar/sidebutton"
	"fiatjaf.com/shiitake/utils"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
	"github.com/nbd-wtf/go-nostr"
)

// Guild is a widget showing a single guild icon.
type Relay struct {
	*sidebutton.Button
	ctx     context.Context
	popover *hoverpopover.MarkupHoverPopover
	url     string
	name    string
}

var guildCSS = cssutil.Applier("guild-guild", `
	.relay-name {
		font-weight: bold;
	}
`)

func NewRelay(ctx context.Context, url string) *Relay {
	g := &Relay{ctx: ctx, url: nostr.NormalizeURL(url)}
	g.Button = sidebutton.NewButton(ctx, func() {
		parent := gtk.BaseWidget(g.Button.Parent())
		parent.ActivateAction("win.open-relay", utils.NewRelayURLVariant(url))
	})

	g.popover = hoverpopover.NewMarkupHoverPopover(g.Button, func(w *hoverpopover.MarkupHoverPopoverWidget) bool {
		w.AddCSSClass("relay-name-popover")
		w.SetPosition(gtk.PosRight)
		w.Label.AddCSSClass("relay-name")
		w.Label.SetText(g.name)
		return true
	})

	g.SetUnavailable()
	guildCSS(g)
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

// SetUnavailable sets the guild as unavailable. It stays unavailable until
// either Invalidate sees it or Update is called on it.
func (g *Relay) SetUnavailable() {
	g.name = "(guild unavailable)"
	g.SetSensitive(false)

	if g.Icon.Initials() == "" {
		g.Icon.SetInitials("?")
	}
}

// Update updates the guild with the given Discord object.
func (g *Relay) Update(url string) {
	g.name = url

	g.SetSensitive(true)

	initials := strings.Split(url, "://")[1]
	g.Icon.SetInitials(initials)
	// g.Icon.SetFromURL() // TODO: get relay icon
}

func (g *Relay) viewChild() {}
