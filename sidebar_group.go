package main

import (
	"context"
	"fmt"

	"fiatjaf.com/nostr-gtk/components/avatar"
	"fiatjaf.com/shiitake/global"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/nbd-wtf/go-nostr/nip29"
)

// Group is a widget showing a single group icon.
type Group struct {
	ctx context.Context

	*gtk.Box

	gad nip29.GroupAddress
}

func NewGroup(ctx context.Context, group *global.Group) *Group {
	g := &Group{
		ctx: ctx,
		gad: group.Address,
	}

	g.Box = gtk.NewBox(gtk.OrientationHorizontal, 0)
	g.SetHExpand(true)

	// indicator := gtk.NewLabel("")
	// indicator.SetHExpand(true)
	// indicator.SetHAlign(gtk.AlignEnd)
	// indicator.SetVAlign(gtk.AlignCenter)

	label := gtk.NewLabel(group.Name)
	label.SetHAlign(gtk.AlignBaseline)
	label.SetHExpand(true)

	if group.Picture != "" {
		icon := avatar.New(ctx, 12, group.Address.String())
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
