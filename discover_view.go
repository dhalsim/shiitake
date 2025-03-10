package main

import (
	"context"
	"time"

	"fiatjaf.com/nostr-gtk/components/avatar"
	"fiatjaf.com/shiitake/global"
	"fiatjaf.com/shiitake/utils"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
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

	current := "groups.fiatjaf.com"

	relayEntry := adw.NewEntryRow()
	relayEntry.SetTitle("relay")
	relayEntry.SetText(current)

	lb := gtk.NewListBox()
	lb.Append(relayEntry)

	relayEntry.ConnectEntryActivated(func() {
		value := relayEntry.Text()
		if value != current {
			current = value
			d.loadRelay(value)
		}
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
		go func() {
			time.Sleep(time.Second * 3)
			glib.IdleAdd(func() {
				win.ErrorToast(err.Error())
			})
		}()
		return
	}

	go func() {
		<-relay.GroupsLoaded

		glib.IdleAdd(func() {
			for lbr := range children[*gtk.ListBox, *gtk.ListBoxRow](d.Results) {
				d.Results.Remove(lbr)
			}

			if relay != nil {
				for _, group := range relay.GroupsList {
					gad := group.Address

					picture := avatar.New(d.ctx, 32, group.Name)
					picture.SetFromURL(group.Picture)
					picture.AddCSSClass("mr-2")

					name := gtk.NewLabel(group.Name)
					name.AddCSSClass("title-3")
					name.AddCSSClass("mb-1")

					id := gtk.NewLabel(group.Address.String())
					id.SetHAlign(gtk.AlignCenter)
					id.AddCSSClass("text-zinc-500")
					id.AddCSSClass("text-xs")

					description := gtk.NewLabel(group.About)
					description.SetWrap(true)

					button := gtk.NewButtonWithLabel("Open")
					button.AddCSSClass("suggested-action")
					button.AddCSSClass("mt-1")
					button.SetHExpand(false)
					button.ConnectClicked(func() {
						revert := utils.ButtonLoading(button, "Opening...")
						glib.IdleAddPriority(glib.PriorityLow, func() {
							win.main.OpenGroup(gad)
							revert()
						})
					})

					grid := gtk.NewGrid()
					grid.AddCSSClass("mx-4")
					grid.AddCSSClass("my-4")
					grid.SetHAlign(gtk.AlignCenter)
					grid.SetHExpand(true)
					grid.Attach(picture /*    */, 0, 0, 1, 3)
					grid.Attach(name /*       */, 1, 0, 4, 1)
					grid.Attach(id /*         */, 1, 1, 4, 1)
					grid.Attach(description /**/, 1, 2, 4, 1)
					grid.Attach(button /*     */, 0, 3, 5, 1)

					d.Results.Append(grid)
				}
			}
		})
	}()
}
