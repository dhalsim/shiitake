package main

import (
	"context"
	"fmt"

	"fiatjaf.com/shiitake/global"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/components/onlineimage"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
	"github.com/diamondburned/gotkit/gtkutil/imgutil"
	"github.com/nbd-wtf/go-nostr/nip29"
)

var _ = cssutil.WriteCSS(`
.groups-viewtree row:hover {
  background: alpha(@theme_fg_color, 0.075);
}
.groups-viewtree row:selected {
  background: alpha(@theme_fg_color, 0.125);
}
.groups-viewtree row:selected:hover {
  background: alpha(@theme_fg_color, 0.175);
}
.group-item {
  padding: 0.35em;
}
.group-item :first-child {
  min-width: 2.5em;
  margin: 0;
}
.group-item expander + * {
  /* Weird workaround because GTK is adding extra padding here for some
   * reason. */
  margin-left: -0.35em;
}
.group-item-muted {
  opacity: 0.35;
}
.group-unread-indicator {
  font-size: 0.75em;
  font-weight: 700;
}
.group-item-unread .group-unread-indicator,
.group-item-mentioned .group-unread-indicator {
  font-size: 0.7em;
  font-weight: 900;
  font-family: monospace;

  min-width: 1em;
  min-height: 1em;
  line-height: 1em;

  padding: 0;
  margin: 0 1em;

  outline: 1.5px solid @theme_fg_color;
  border-radius: 99px;
}
.group-item-mentioned .group-unread-indicator {
  font-size: 0.8em;
  outline-color: @mentioned;
  background: @mentioned;
  color: @theme_bg_color;
}
`)

// Group is a widget showing a single group icon.
type Group struct {
	ctx context.Context

	*gtk.Box

	gad nip29.GroupAddress
}

var groupCSS = cssutil.Applier("group-group", `
.group-name {
  font-weight: bold;
}
`)

func NewGroup(ctx context.Context, group *global.Group) *Group {
	g := &Group{
		ctx: ctx,
		gad: group.Address,
	}

	g.Box = gtk.NewBox(gtk.OrientationHorizontal, 0)
	g.SetHExpand(true)

	// indicator := gtk.NewLabel("")
	// indicator.AddCSSClass("group-unread-indicator")
	// indicator.SetHExpand(true)
	// indicator.SetHAlign(gtk.AlignEnd)
	// indicator.SetVAlign(gtk.AlignCenter)

	label := gtk.NewLabel(group.Name)
	label.AddCSSClass("group-item")
	label.SetHAlign(gtk.AlignBaseline)
	label.SetHExpand(true)

	groupCSS(g)

	if group.Picture != "" {
		icon := onlineimage.NewAvatar(ctx, imgutil.HTTPProvider, 12)
		icon.SetFromURL(group.Picture)
		g.Box.Append(icon)
	}

	g.Box.Append(label)
	// g.Box.Append(indicator)

	go func() {
		for {
			select {
			case <-group.GroupUpdated:
				button := g.Box.LastChild().(*gtk.Label)
				button.SetText(group.Name)
			case err := <-group.NewError:
				fmt.Println(group.Address, "ERROR", err)
			}
		}
	}()

	return g
}
