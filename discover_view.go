package main

import (
	"context"

	"fiatjaf.com/nostr-gtk/components/avatar"
	"fiatjaf.com/shiitake/global"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type DiscoverView struct {
	*gtk.Box
	ctx context.Context

	Results *gtk.ListBox
}

func NewDiscoverView(ctx context.Context) *DiscoverView {
	d := &DiscoverView{
		Box: gtk.NewBox(gtk.OrientationVertical, 0),
		ctx: ctx,
	}

	relayEntry := adw.NewEntryRow()
	relayEntry.SetTitle("relay")
	relayEntry.SetText("groups.fiatjaf.com")

	lb := gtk.NewListBox()
	lb.Append(relayEntry)

	relayEntry.ConnectEntryActivated(func() {
		d.loadRelay(relayEntry.Text())
	})
	d.Append(lb)

	d.loadRelay(relayEntry.Text())

	d.Results = gtk.NewListBox()

	scrolledWindow := gtk.NewScrolledWindow()
	scrolledWindow.SetHExpand(true)
	scrolledWindow.SetVExpand(true)

	scrolledWindow.SetChild(d.Results)
	d.Append(scrolledWindow)

	return d
}

func (d *DiscoverView) loadRelay(url string) {
	relay, err := global.LoadRelay(d.ctx, url)
	if err != nil {
		win.ErrorToast(err.Error())
		return
	}

	go func() {
		<-relay.GroupsLoaded

		glib.IdleAdd(func() {
			eachChild(d.Results, func(lbr *gtk.ListBoxRow) bool {
				d.Results.Remove(lbr)
				return false
			})

			if relay != nil {
				for _, group := range relay.GroupsList {
					gad := group.Address

					picture := avatar.New(d.ctx, 32, group.Name)
					picture.SetFromURL(group.Picture)
					picture.AddCSSClass("mr-2")

					name := gtk.NewLabel(group.Name)
					name.AddCSSClass("title-3")
					name.AddCSSClass("mb-1")

					description := gtk.NewLabel(group.About)
					description.SetWrap(true)

					button := gtk.NewButtonWithLabel("Open")
					button.AddCSSClass("suggested-action")
					button.AddCSSClass("mt-1")
					button.SetHExpand(false)
					button.ConnectClicked(func() {
						button.SetLabel("Opening...")
						button.SetSensitive(false)
						button.RemoveCSSClass("suggested-action")

						go func() {
							win.main.OpenGroup(gad)

							button.SetLabel("Open")
							button.SetSensitive(true)
							button.AddCSSClass("suggested-action")
						}()
					})

					grid := gtk.NewGrid()
					grid.AddCSSClass("mx-4")
					grid.AddCSSClass("my-4")
					grid.SetHAlign(gtk.AlignCenter)
					grid.SetHExpand(true)
					grid.Attach(picture /*    */, 0, 0, 1, 4)
					grid.Attach(name /*       */, 1, 0, 4, 1)
					grid.Attach(description /**/, 1, 1, 4, 3)
					grid.Attach(button /*     */, 0, 4, 5, 1)

					d.Results.Append(grid)
				}
			}
		})
	}()
}
