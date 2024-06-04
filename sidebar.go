package main

import (
	"context"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
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

	s.GroupsView = NewGroupsView(s.ctx)
	s.GroupsView.List.GrabFocus()
	s.GroupsView.SetVExpand(true)
	s.GroupsView.SetSizeRequest(100, -1)

	userBar := NewUserBar(ctx)

	sep := gtk.NewSeparator(gtk.OrientationVertical)
	sep.AddCSSClass("spacer")
	sep.SetVExpand(true)

	box := gtk.NewBox(gtk.OrientationVertical, 0)
	box.SetName("sidebar-box")
	box.SetHAlign(gtk.AlignFill)
	box.SetSizeRequest(100, -1)
	box.SetVExpand(true)
	box.SetHExpand(true)
	box.Append(s.GroupsView)
	box.Append(sep)
	box.Append(userBar)

	s.ScrolledWindow = gtk.NewScrolledWindow()
	s.ScrolledWindow.SetName("sidebar")
	s.ScrolledWindow.SetChild(box)
	s.ScrolledWindow.SetHExpand(true)
	s.ScrolledWindow.SetHAlign(gtk.AlignFill)

	return &s
}
