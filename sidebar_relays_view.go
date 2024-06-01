package main

import (
	"context"
	"strings"

	"fiatjaf.com/shiitake/components/sidebutton"
	"fiatjaf.com/shiitake/global"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
)

// View contains a list of relays and folders.
type RelaysView struct {
	Widget *gtk.ListBox

	ctx context.Context
}

var relaysViewCSS = cssutil.Applier("relay-view", `
.relay-view {
  margin: 4px 0;
}
.relay-view button:active:not(:hover) {
  background: initial;
}
`)

func NewRelaysView(ctx context.Context) *RelaysView {
	v := RelaysView{
		ctx: ctx,
	}

	v.Widget = gtk.NewListBox()
	relaysViewCSS(v.Widget)

	go func() {
		me := global.GetMe(ctx)
		for {
			select {
			case relay := <-me.JoinedRelay:
				g := NewRelayButton(v.ctx, relay)
				v.Widget.Prepend(g)
			case url := <-me.LeftRelay:
				row := getChild(v.Widget, func(lbr *gtk.ListBoxRow) bool { return url == lbr.Name() })
				if row != nil {
					v.Widget.Remove(row)
				}
			}
		}
	}()

	return &v
}

type RelayButton struct {
	*sidebutton.Button
	ctx  context.Context
	url  string
	name string
}

func NewRelayButton(ctx context.Context, relay *global.Relay) *RelayButton {
	g := &RelayButton{
		ctx:  ctx,
		url:  relay.URL,
		name: relay.Name,
	}

	g.name = relay.URL

	g.Button = sidebutton.NewButton(ctx, func() {
		win.OpenRelay(relay.URL)
	})

	g.SetTooltipMarkup(trimProtocol(relay.URL))

	g.SetSensitive(true)
	initials := strings.Join(strings.Split(strings.Split(relay.URL, "://")[1], "."), " ")
	g.Icon.SetInitials(initials)
	if relay.Image != "" {
		g.Icon.SetFromURL(relay.Image)
	}

	return g
}
