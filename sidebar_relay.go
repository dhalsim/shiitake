package main

import (
	"context"
	"strings"

	"fiatjaf.com/shiitake/components/hoverpopover"
	"fiatjaf.com/shiitake/components/sidebutton"
	"fiatjaf.com/shiitake/global"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
)

// Relay is a widget showing a single relay icon.
type Relay struct {
	*sidebutton.Button
	ctx  context.Context
	url  string
	name string
}

var relayCSS = cssutil.Applier("relay-relay", `
.relay-name {
  font-weight: bold;
}
`)

func NewRelay(ctx context.Context, relay *global.Relay) *Relay {
	g := &Relay{
		ctx:  ctx,
		url:  relay.URL,
		name: relay.Name,
	}

	g.name = relay.URL

	g.Button = sidebutton.NewButton(ctx, func() {
		win.OpenRelay(relay.URL)
	})

	hoverpopover.NewMarkupHoverPopover(g.Button, func(w *hoverpopover.MarkupHoverPopoverWidget) bool {
		w.AddCSSClass("relay-name-popover")
		w.SetPosition(gtk.PosRight)
		w.Label.AddCSSClass("relay-name")
		w.Label.SetText(g.name)
		return true
	})

	relayCSS(g)

	g.SetSensitive(true)
	initials := strings.Join(strings.Split(strings.Split(relay.URL, "://")[1], "."), " ")
	g.Icon.SetInitials(initials)
	if relay.Image != "" {
		g.Icon.SetFromURL(relay.Image)
	}

	return g
}

func (g *Relay) viewChild() {}
