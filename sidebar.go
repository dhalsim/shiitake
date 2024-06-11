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
	discover.AddCSSClass("frame")
	discover.AddCSSClass("border-2")

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

				// if we have just asked to join this group, we do this so we reload it
				if win.main.Groups.currentGroup() != nil && win.main.Groups.currentGroup().Address.Equals(gad) {
					win.main.Groups.switchTo(nip29.GroupAddress{})
				}

				glib.IdleAdd(func() {
					button := sidebutton.New(ctx, group.Name, func() {
						win.main.OpenGroup(gad)
					})

					lbr := gtk.NewListBoxRow()
					lbr.SetName(gad.String())
					lbr.SetChild(button)

					groupsList.Append(lbr)

					group.OnUpdated(func() {
						button.Label.SetText(group.Name)
						button.Icon.SetFromURL(group.Picture)
					})
				})
			case gad := <-me.LeftGroup:
				eachChild(groupsList, func(lbr *gtk.ListBoxRow) bool {
					if lbr.Name() == gad.String() {
						groupsList.Remove(lbr)
						return true // stop
					}
					return false // continue
				})
			}
		}
	}()

	s.selectGroup = func(gad nip29.GroupAddress) {
		if gad.IsValid() {
			discover.RemoveCSSClass("frame")
			discover.RemoveCSSClass("border-2")
		} else {
			discover.AddCSSClass("frame")
			discover.AddCSSClass("border-2")
		}

		eachChild(groupsList, func(lbr *gtk.ListBoxRow) bool {
			// iterate through all buttons, removing classes from all and adding in the selected
			sidebuttonWidget := lbr.Child().(*gtk.Button)
			if lbr.Name() == gad.String() {
				sidebuttonWidget.AddCSSClass("frame")
				sidebuttonWidget.AddCSSClass("border-2")
			} else {
				sidebuttonWidget.RemoveCSSClass("frame")
				sidebuttonWidget.RemoveCSSClass("border-2")
			}

			return false
		})
	}

	return s
}
