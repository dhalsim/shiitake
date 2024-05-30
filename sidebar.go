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

	current *GroupsView

	Left   *gtk.Box
	AddNew *AddRelayButton
	Relays *RelaysView
	Right  *gtk.Stack

	placeholder gtk.Widgetter

	ctx context.Context
}

func NewSidebar(ctx context.Context) *Sidebar {
	s := Sidebar{
		ctx: ctx,
	}

	s.Relays = NewRelaysView(ctx)

	s.AddNew = NewAddRelayButton(ctx, func(v string) {
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
	leftBox.Append(s.Relays)
	leftBox.Append(separator)
	leftBox.Append(s.AddNew)

	leftScroll := gtk.NewScrolledWindow()
	leftScroll.SetVExpand(true)
	leftScroll.SetPolicy(gtk.PolicyNever, gtk.PolicyExternal)
	leftScroll.SetChild(leftBox)

	leftCtrl := gtk.NewWindowControls(gtk.PackStart)
	leftCtrl.SetHAlign(gtk.AlignCenter)

	s.Left = gtk.NewBox(gtk.OrientationVertical, 0)
	s.Left.AddCSSClass("sidebar-relayside")
	s.Left.Append(leftCtrl)
	s.Left.Append(leftScroll)

	s.placeholder = gtk.NewWindowHandle()

	s.Right = gtk.NewStack()
	s.Right.SetSizeRequest(groupsWidth, -1)
	s.Right.SetVExpand(true)
	s.Right.SetHExpand(true)
	s.Right.AddChild(s.placeholder)
	s.Right.SetVisibleChild(s.placeholder)
	s.Right.SetTransitionType(gtk.StackTransitionTypeCrossfade)

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
	rightWrap.SetContent(s.Right)

	s.Box = gtk.NewBox(gtk.OrientationHorizontal, 0)
	s.Box.SetHExpand(false)
	s.Box.Append(s.Left)
	s.Box.Append(rightWrap)
	sidebarCSS(s)

	return &s
}

func (s *Sidebar) openRelay(relayURL string) {
	if s.current != nil && s.current.RelayURL == relayURL {
		// we're already there.
		return
	}

	chs := NewGroupsView(s.ctx, relayURL)

	s.current = chs

	chs.SetVExpand(true)
	s.current = chs

	s.Right.AddChild(chs)
	s.Right.SetVisibleChild(chs)

	chs.List.GrabFocus()
}
