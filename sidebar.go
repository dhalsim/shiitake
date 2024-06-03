package main

import (
	"context"

	"fiatjaf.com/shiitake/global"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/nbd-wtf/go-nostr/nip29"
)

type Sidebar struct {
	*gtk.Box // horizontal

	RelaysView        *RelaysView
	CurrentGroupsView *GroupsView

	ctx context.Context

	openRelay func(relayURL string)
}

func NewSidebar(ctx context.Context) *Sidebar {
	s := Sidebar{
		ctx: ctx,
	}

	s.RelaysView = NewRelaysView(ctx)

	addNew := NewAddRelayButton(ctx, func(v string) {
		if v != "" {
			gad, err := nip29.ParseGroupAddress(v)
			if err != nil {
				// we only accept full group identifiers for now (TODO)
				return
			}
			global.JoinGroup(ctx, gad)
		}
	})

	separator := gtk.NewSeparator(gtk.OrientationHorizontal)

	// leftBox holds just the new button and the relay view, as opposed to s.Left
	// which holds the scrolled window and the window controls.
	leftBox := gtk.NewBox(gtk.OrientationVertical, 0)
	leftBox.Append(s.RelaysView.Widget)
	leftBox.Append(separator)
	leftBox.Append(addNew)

	leftScroll := gtk.NewScrolledWindow()
	leftScroll.SetVExpand(true)
	leftScroll.SetPolicy(gtk.PolicyNever, gtk.PolicyExternal)
	leftScroll.SetChild(leftBox)

	leftCtrl := gtk.NewWindowControls(gtk.PackStart)
	leftCtrl.SetHAlign(gtk.AlignCenter)

	left := gtk.NewBox(gtk.OrientationVertical, 0)
	left.Append(leftCtrl)
	left.Append(leftScroll)

	windowHandle := gtk.NewWindowHandle()
	windowHandle.SetChild(gtk.NewLabel(""))

	groupsViewStack := gtk.NewStack()
	groupsViewStack.SetSizeRequest(groupsWidth, -1)
	groupsViewStack.SetVExpand(true)
	groupsViewStack.SetHExpand(true)
	groupsViewStack.AddChild(windowHandle)
	groupsViewStack.SetVisibleChild(windowHandle)
	groupsViewStack.SetTransitionType(gtk.StackTransitionTypeCrossfade)

	s.openRelay = func(relayURL string) {
		windowHandle.Child().(*gtk.Label).SetText(trimProtocol(relayURL))

		if s.CurrentGroupsView != nil && s.CurrentGroupsView.RelayURL == relayURL {
			// we're already there.
			return
		}

		existing := groupsViewStack.ChildByName(relayURL)
		if existing != nil {
			// this is already initialized
			groupsViewStack.SetVisibleChild(existing)
		} else {
			// otherwise we initialize it and add
			groupsView := NewGroupsView(s.ctx, relayURL)
			groupsView.SetName(relayURL)
			groupsView.SetVExpand(true)

			groupsViewStack.AddChild(groupsView)
			groupsViewStack.SetVisibleChild(groupsView)
			s.CurrentGroupsView = groupsView

			groupsView.List.GrabFocus()
		}
	}

	userBar := newUserBar(ctx)

	rightWrap := adw.NewToolbarView()
	rightWrap.AddBottomBar(userBar)
	rightWrap.SetContent(groupsViewStack)

	s.Box = gtk.NewBox(gtk.OrientationHorizontal, 0)
	s.Box.SetHExpand(false)
	s.Box.Append(left)
	s.Box.Append(rightWrap)

	return &s
}
