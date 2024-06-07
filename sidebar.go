package main

import (
	"context"
	"fmt"

	"fiatjaf.com/shiitake/components/sidebutton"
	"fiatjaf.com/shiitake/global"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/gtkutil"
	"github.com/nbd-wtf/go-nostr/nip29"
)

type Sidebar struct {
	*gtk.ScrolledWindow

	GroupsView *GroupsView

	ctx context.Context
}

func NewSidebar(ctx context.Context) *Sidebar {
	s := Sidebar{
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

	s.GroupsView = NewGroupsView(s.ctx)
	s.GroupsView.List.GrabFocus()
	s.GroupsView.SetVExpand(true)
	s.GroupsView.SetSizeRequest(100, -1)

	userBar := NewUserBar(ctx)

	box := gtk.NewBox(gtk.OrientationVertical, 0)
	box.SetName("sidebar-box")
	box.SetHAlign(gtk.AlignFill)
	box.SetSizeRequest(100, -1)
	box.SetVExpand(true)
	box.SetHExpand(true)
	box.Append(discover)
	box.Append(sep1)
	box.Append(s.GroupsView)
	box.Append(sep2)
	box.Append(userBar)

	s.ScrolledWindow = gtk.NewScrolledWindow()
	s.ScrolledWindow.SetName("sidebar")
	s.ScrolledWindow.SetChild(box)
	s.ScrolledWindow.SetHExpand(true)
	s.ScrolledWindow.SetHAlign(gtk.AlignFill)

	return &s
}

type GroupsView struct {
	*gtk.ScrolledWindow
	List *gtk.ListBox

	ctx gtkutil.Cancellable

	selectGAD nip29.GroupAddress // delegate to select later
}

func NewGroupsView(ctx context.Context) *GroupsView {
	v := GroupsView{}

	var current nip29.GroupAddress

	v.List = gtk.NewListBox()
	v.List.SetName("groups-list")
	v.List.SetSelectionMode(gtk.SelectionNone)
	v.List.SetHExpand(true)
	v.List.SetVExpand(true)
	v.List.ConnectRowSelected(func(row *gtk.ListBoxRow) {
		gad, _ := nip29.ParseGroupAddress(row.Name())
		if gad.Equals(current) {
			return
		}
		win.main.OpenGroup(gad)
	})

	go func() {
		me := global.GetMe(ctx)
		for {
			select {
			case group := <-me.JoinedGroup:
				gad := group.Address

				glib.IdleAdd(func() {
					button := sidebutton.New(ctx, group.Name, func() {
						if gad.Equals(current) {
							return
						}
						win.main.OpenGroup(gad)
					})

					lbr := gtk.NewListBoxRow()
					lbr.SetName(gad.String())
					lbr.SetChild(button)

					v.List.Append(lbr)
					win.main.OpenGroup(group.Address)

					go func() {
						for {
							select {
							case <-group.GroupUpdated:
								button.Label.SetText(group.Name)
								button.Icon.SetFromURL(group.Picture)
								if win.main.Messages.currentGroup.Address.Equals(group.Address) {
									win.main.Header.SetTitleWidget(adw.NewWindowTitle(group.Name, group.Address.String()))
								}
							case err := <-group.NewError:
								fmt.Println(group.Address, "ERROR", err)
							}
						}
					}()
				})
			case gad := <-me.LeftGroup:
				eachChild(v.List, func(lbr *gtk.ListBoxRow) bool {
					if lbr.Name() == gad.String() {
						glib.IdleAdd(func() {
							v.List.Remove(lbr)
						})
						return true // stop
					}
					return false // continue
				})
			}
		}
	}()

	v.ScrolledWindow = gtk.NewScrolledWindow()
	v.ScrolledWindow.SetName("groups-view")
	v.ScrolledWindow.SetVExpand(true)
	v.ScrolledWindow.SetVAlign(gtk.AlignFill)
	v.ScrolledWindow.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	v.ScrolledWindow.SetChild(v.List)

	// bind the context to cancel when we're hidden.
	v.ctx = gtkutil.WithVisibility(ctx, v)

	return &v
}
