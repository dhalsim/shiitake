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

	userBar := newUserBar(ctx)
	userBar.SetVAlign(gtk.AlignEnd)

	sep := gtk.NewSeparator(gtk.OrientationVertical)
	sep.AddCSSClass("spacer")
	sep.SetVExpand(true)

	box := gtk.NewBox(gtk.OrientationVertical, 0)
	box.SetSizeRequest(100, -1)
	box.SetVExpand(true)
	box.SetHExpand(true)
	box.Append(s.GroupsView)
	box.Append(sep)
	box.Append(userBar)

	s.ScrolledWindow = gtk.NewScrolledWindow()
	s.ScrolledWindow.SetChild(box)

	return &s
}
