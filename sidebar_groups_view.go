package main

import (
	"context"

	"fiatjaf.com/shiitake/global"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gotkit/gtkutil"
	"github.com/nbd-wtf/go-nostr/nip29"
)

const groupsWidth = 300

type GroupsView struct {
	*adw.ToolbarView

	Header struct {
		*adw.HeaderBar
		Name *gtk.Label
	}

	Scroll *gtk.ScrolledWindow
	List   *gtk.ListBox

	ctx gtkutil.Cancellable

	RelayURL  string
	selectGAD nip29.GroupAddress // delegate to select later
}

func NewGroupsView(ctx context.Context, relayURL string) *GroupsView {
	v := GroupsView{
		RelayURL: relayURL,
	}

	v.ToolbarView = adw.NewToolbarView()
	v.ToolbarView.SetTopBarStyle(adw.ToolbarFlat)
	v.ToolbarView.SetExtendContentToTopEdge(true) // basically act like an overlay

	// Bind the context to cancel when we're hidden.
	v.ctx = gtkutil.WithVisibility(ctx, v)

	v.Header.Name = gtk.NewLabel("")
	v.Header.Name.SetHAlign(gtk.AlignStart)
	v.Header.Name.SetEllipsize(pango.EllipsizeEnd)

	// The header is placed on top of the overlay, kind of like the official
	// client.
	v.Header.HeaderBar = adw.NewHeaderBar()
	v.Header.HeaderBar.SetShowTitle(false)
	v.Header.HeaderBar.PackStart(v.Header.Name)

	viewport := gtk.NewViewport(nil, nil)

	v.Scroll = gtk.NewScrolledWindow()
	v.Scroll.SetVExpand(true)
	v.Scroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	v.Scroll.SetChild(viewport)
	// v.Scroll.SetPropagateNaturalWidth(true)
	// v.Scroll.SetPropagateNaturalHeight(true)

	var current nip29.GroupAddress

	v.List = gtk.NewListBox()
	v.List.SetSelectionMode(gtk.SelectionSingle)
	v.List.SetSizeRequest(groupsWidth, -1)
	v.List.SetHExpand(true)
	v.List.SetVExpand(true)
	v.List.ConnectRowSelected(func(row *gtk.ListBoxRow) {
		gad, _ := nip29.ParseGroupAddress(row.Name())
		if gad.Equals(current) {
			return
		}
		win.OpenGroup(gad)
	})

	go func() {
		me := global.GetMe(ctx)
		for {
			select {
			case group := <-me.JoinedGroup:
				glib.IdleAdd(func() {
					g := NewGroup(ctx, group)
					lbr := gtk.NewListBoxRow()
					lbr.SetName(g.gad.String())
					lbr.SetChild(g)
					v.List.Append(lbr)
					win.OpenGroup(group.Address)
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

	viewport.SetChild(v.List)
	viewport.SetFocusChild(v.List)

	v.ToolbarView.AddTopBar(v.Header)
	v.ToolbarView.SetContent(v.Scroll)
	v.ToolbarView.SetFocusChild(v.Scroll)

	return &v
}

type GroupsListView struct {
	*gtk.Box
	Children []*Group
}

func (v *GroupsListView) get(needle nip29.GroupAddress) *Group {
	for _, child := range v.Children {
		if child.gad.Equals(needle) {
			return child
		}
	}
	return nil
}

func (v *GroupsListView) append(this *Group) {
	v.Children = append(v.Children, this)
	v.Box.Append(this)
}

func (v *GroupsListView) remove(this nip29.GroupAddress) {
	for i, child := range v.Children {
		if child.gad.Equals(this) {
			v.Children = append(v.Children[:i], v.Children[i+1:]...)
			v.Box.Remove(child)
			break
		}
	}
}
