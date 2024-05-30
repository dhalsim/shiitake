package main

import (
	"context"

	"fiatjaf.com/shiitake/global"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/gtkutil"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
	"github.com/nbd-wtf/go-nostr/nip29"
)

var sidebarCSS = cssutil.Applier("sidebar-sidebar", `
	@define-color sidebar_bg mix(@borders, @theme_bg_color, 0.25);

	windowcontrols.end:not(.empty) {
		margin-right: 4px;
	}
	windowcontrols.start:not(.empty) {
		margin: 4px;
		margin-right: 0;
	}
	.sidebar-relayside {
		background-color: @sidebar_bg;
	}
`)

type Sidebar struct {
	*gtk.Box // horizontal

	RelaysView        *RelaysView
	CurrentGroupsView *GroupsView

	placeholder gtk.Widgetter

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
	separator.AddCSSClass("sidebar-separator")

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
	left.AddCSSClass("sidebar-relayside")
	left.Append(leftCtrl)
	left.Append(leftScroll)

	s.placeholder = gtk.NewWindowHandle()

	groupsViewStack := gtk.NewStack()
	groupsViewStack.SetSizeRequest(groupsWidth, -1)
	groupsViewStack.SetVExpand(true)
	groupsViewStack.SetHExpand(true)
	groupsViewStack.AddChild(s.placeholder)
	groupsViewStack.SetVisibleChild(s.placeholder)
	groupsViewStack.SetTransitionType(gtk.StackTransitionTypeCrossfade)

	s.openRelay = func(relayURL string) {
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

	userBar := newUserBar(ctx, []gtkutil.PopoverMenuItem{
		gtkutil.MenuItem("Quick Switcher", "win.quick-switcher"),
		gtkutil.MenuSeparator("User Settings"),
		gtkutil.Submenu("Set _Status", []gtkutil.PopoverMenuItem{
			gtkutil.MenuItem("_Online", "win.set-online"),
			gtkutil.MenuItem("_Idle", "win.set-idle"),
			gtkutil.MenuItem("_Do Not Disturb", "win.set-dnd"),
			gtkutil.MenuItem("In_visible", "win.set-invisible"),
		}),
		gtkutil.MenuSeparator(""),
		gtkutil.MenuItem("_Preferences", "app.preferences"),
		gtkutil.MenuItem("_About", "app.about"),
		gtkutil.MenuItem("_Logs", "app.logs"),
		gtkutil.MenuItem("_Quit", "app.quit"),
	})

	// TODO: consider if we can merge this ToolbarView with the one in groups
	// and direct.
	rightWrap := adw.NewToolbarView()
	rightWrap.AddBottomBar(userBar)
	rightWrap.SetContent(groupsViewStack)

	s.Box = gtk.NewBox(gtk.OrientationHorizontal, 0)
	s.Box.SetHExpand(false)
	s.Box.Append(left)
	s.Box.Append(rightWrap)
	sidebarCSS(s)

	return &s
}
