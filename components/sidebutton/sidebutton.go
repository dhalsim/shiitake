package sidebutton

import (
	"context"

	"fiatjaf.com/nostr-gtk/components/avatar"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type Sidebutton struct {
	*gtk.Button
	ctx context.Context

	Icon  *avatar.Avatar
	Label *gtk.Label
}

func New(ctx context.Context, label string, open func()) *Sidebutton {
	g := &Sidebutton{
		ctx: ctx,
	}

	g.Icon = avatar.New(ctx, 18, "")
	g.Icon.AddCSSClass("mr-2")

	g.Label = gtk.NewLabel(label)

	box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	box.Append(g.Icon)
	box.Append(g.Label)

	g.Button = gtk.NewButton()
	g.Button.SetHasFrame(false)
	g.Button.SetHAlign(gtk.AlignFill)
	g.Button.SetHExpand(true)
	g.Button.SetChild(box)
	g.Button.AddCSSClass("mx-2")
	g.Button.AddCSSClass("my-1")
	g.Button.ConnectClicked(open)

	return g
}
