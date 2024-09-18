package main

import (
	"context"

	"fiatjaf.com/shiitake/components/sidebutton"
	"fiatjaf.com/shiitake/global"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/nbd-wtf/go-nostr/nip29"
)

type Sidebar struct {
	*gtk.ScrolledWindow

	ctx context.Context

	selectGroup func(nip29.GroupAddress)
}

func NewSidebar(ctx context.Context) *Sidebar {
	s := &Sidebar{
		ctx: ctx,
	}

	discover := sidebutton.New(ctx, "Discover", func() {
		win.main.OpenDiscover()
	})
	discover.Icon.Avatar.SetIconName("earth-symbolic")

	sep1 := gtk.NewSeparator(gtk.OrientationVertical)
	sep1.AddCSSClass("spacer")

	sep2 := gtk.NewSeparator(gtk.OrientationVertical)
	sep2.AddCSSClass("spacer")

	sep3 := gtk.NewSeparator(gtk.OrientationVertical)
	sep3.AddCSSClass("spacer")

	groupsList := gtk.NewListBox()
	groupsList.SetName("groups-list")
	groupsList.SetSelectionMode(gtk.SelectionNone)
	groupsList.SetHExpand(true)
	groupsList.SetVExpand(true)
	groupsList.GrabFocus()

	groupsScroll := gtk.NewScrolledWindow()
	groupsScroll.SetName("groups-view")
	groupsScroll.SetSizeRequest(150, -1)
	groupsScroll.SetVExpand(true)
	groupsScroll.SetVAlign(gtk.AlignFill)
	groupsScroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	groupsScroll.SetChild(groupsList)

	userBar := NewUserBar(ctx)

	box := gtk.NewBox(gtk.OrientationVertical, 0)
	box.SetName("sidebar-box")
	box.SetHAlign(gtk.AlignFill)
	box.SetSizeRequest(100, -1)
	box.SetVExpand(true)
	box.SetHExpand(true)
	box.Append(sep1)
	box.Append(discover)
	box.Append(sep2)
	box.Append(groupsScroll)
	box.Append(sep3)
	box.Append(userBar)

	s.ScrolledWindow = gtk.NewScrolledWindow()
	s.ScrolledWindow.SetName("sidebar")
	s.ScrolledWindow.SetChild(box)
	s.ScrolledWindow.SetHExpand(true)
	s.ScrolledWindow.SetHAlign(gtk.AlignFill)

	go func() {
		me := global.GetMe(ctx)
		for {
			select {
			case group := <-me.JoinedGroup:
				gad := group.Address
				glib.IdleAdd(func() {
					button := sidebutton.New(ctx, group.Name, func() {
						win.main.OpenGroup(gad)
					})

					lbr := gtk.NewListBoxRow()
					lbr.SetName(gad.String())
					lbr.SetChild(button)

					groupsList.Append(lbr)

					group.OnUpdated(func() {
						glib.IdleAdd(func() {
							button.Label.SetText(group.Name)
							button.Icon.SetFromURL(group.Picture)
						})
					})
				})
			case gad := <-me.LeftGroup:
				glib.IdleAdd(func() {
					for lbr := range children[*gtk.ListBox, *gtk.ListBoxRow](groupsList) {
						if lbr.Name() == gad.String() {
							groupsList.Remove(lbr)
							break
						}
					}
				})
			}
		}
	}()

	s.selectGroup = func(gad nip29.GroupAddress) {
		glib.IdleAddPriority(glib.PriorityLow, func() {
			if gad.IsValid() {
				discover.RemoveCSSClass("bg-amber-400")
			} else {
				discover.AddCSSClass("bg-amber-400")
			}

			for lbr := range children[*gtk.ListBox, *gtk.ListBoxRow](groupsList) {
				// iterate through all buttons, removing classes from all and adding in the selected
				sidebuttonWidget := lbr.Child().(*gtk.Button)
				if lbr.Name() == gad.String() {
					sidebuttonWidget.AddCSSClass("bg-amber-400")
				} else {
					sidebuttonWidget.RemoveCSSClass("bg-amber-400")
				}
			}
		})
	}

	return s
}
