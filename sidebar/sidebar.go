// Package sidebar contains the sidebar showing relays and groups.
package sidebar

import (
	"context"

	"fiatjaf.com/shiitake/global"
	"fiatjaf.com/shiitake/sidebar/groups"
	"fiatjaf.com/shiitake/sidebar/relays"
	"fiatjaf.com/shiitake/utils"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/gtkutil"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
	"github.com/nbd-wtf/go-nostr/nip29"
)

// Sidebar is the bar on the left side of the application once it's logged in.
type Sidebar struct {
	*gtk.Box // horizontal

	Left   *gtk.Box
	AddNew *relays.Button
	Relays *relays.View
	Right  *gtk.Stack

	// Keep track of the last child to remove.
	current struct {
		w gtk.Widgetter
		// id string
	}
	placeholder gtk.Widgetter

	ctx context.Context
}

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

// NewSidebar creates a new Sidebar.
func NewSidebar(ctx context.Context) *Sidebar {
	s := Sidebar{
		ctx: ctx,
	}

	s.Relays = relays.NewView(ctx)
	// s.Relays.Invalidate()

	s.AddNew = relays.NewButton(ctx, func(v string) {
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
	s.Right.SetSizeRequest(groups.GroupsWidth, -1)
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

// RelayID returns the relay ID that the group list is showing for, if any.
// If not, 0 is returned.
func (s *Sidebar) RelayID() string {
	ch, ok := s.current.w.(*groups.View)
	if !ok {
		return ""
	}
	return ch.RelayID()
}

func (s *Sidebar) removeCurrent() {
	if s.current.w == nil {
		return
	}

	w := s.current.w
	s.current.w = nil

	if w == nil {
		return
	}

	gtkutil.NotifyProperty(s.Right, "transition-running", func() bool {
		// Remove the widget when the transition is done.
		if !s.Right.TransitionRunning() {
			s.Right.Remove(w)
			return true
		}
		return false
	})
}

func (s *Sidebar) openRelay(relayID string) *groups.View {
	chs, ok := s.current.w.(*groups.View)
	if ok && chs.RelayID() == relayID {
		// We're already there.
		return chs
	}

	s.unselect()
	s.Relays.SetSelectedRelay(relayID)

	chs = groups.NewView(s.ctx, relayID)
	chs.SetVExpand(true)
	s.current.w = chs

	s.Right.AddChild(chs)
	s.Right.SetVisibleChild(chs)

	chs.Child.View.GrabFocus()
	chs.InvalidateHeader()
	return chs
}

func (s *Sidebar) unselect() {
	s.Relays.Unselect()
	s.removeCurrent()
}

// Unselect unselects the current relay or group.
func (s *Sidebar) Unselect() {
	s.unselect()
	s.Right.SetVisibleChild(s.placeholder)
}

// SetSelectedRelay marks the relay with the given ID as selected.
func (s *Sidebar) SetSelectedRelay(relayURL string) {
	s.Relays.SetSelectedRelay(relayURL)
	s.openRelay(relayURL)
}

// SelectRelay selects and activates the relay with the given ID.
func (s *Sidebar) SelectRelay(url string) {
	if s.Relays.SelectedRelayURL() != url {
		s.Relays.SetSelectedRelay(url)

		parent := gtk.BaseWidget(s.Parent())
		parent.ActivateAction("win.open-relay", utils.NewRelayURLVariant(url))
	}
}

// SelectGroup selects and activates the group with the given ID. It ensures
// that the sidebar is at the right place then activates the controller.
// This function acts the same as if the user clicked on the group, meaning it
// funnels down to a single widget that then floats up to the controller.
func (s *Sidebar) SelectGroup(chID string) {
	// state := gtkcord.FromContext(s.ctx)
	// ch, _ := state.Cabinet.Group(chID)
	// if ch == nil {
	// 	log.Println("sidebar: group with ID", chID, "not found")
	// 	return
	// }

	// s.Relays.SetSelectedRelay(ch.RelayID)

	// if ch.RelayID.IsValid() {
	// 	relay := s.openRelay(ch.RelayID)
	// 	relay.SelectGroup(chID)
	// } else {
	// 	direct := s.OpenDMs()
	// 	direct.SelectGroup(chID)
	// }
}
