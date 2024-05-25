package window

import (
	"context"
	"fmt"

	"fiatjaf.com/shiitake/global"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gotkit/app"
	"github.com/diamondburned/gotkit/gtkutil"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
	"github.com/nbd-wtf/go-nostr/nip29"
)

const GroupsWidth = bannerWidth

type GroupsView struct {
	*adw.ToolbarView

	Header struct {
		*adw.HeaderBar
		Name *gtk.Label
	}

	Scroll *gtk.ScrolledWindow
	Child  *GroupsListView

	ctx gtkutil.Cancellable

	RelayURL  string
	selectGAD nip29.GroupAddress // delegate to select later
}

var viewCSS = cssutil.Applier("groups-view", `
	.groups-viewtree {
		background: none;
	}
	/* GTK is dumb. There's absolutely no way to get a ListItemWidget instance
	 * to style it, so we'll just unstyle everything and use the child instead.
	 */
	.groups-viewtree > row {
		margin: 0;
		padding: 0;
	}
	.groups-header {
		padding: 0 {$header_padding};
		border-radius: 0;
	}
	.groups-view-scroll {
		/* Space out the header, since it's in an overlay. */
		margin-top: {$header_height};
	}
	.groups-has-banner .groups-view-scroll {
		/* No need to space out here, since we have the banner. We do need to
		 * turn the header opaque with the styling below though, so the user can
		 * see it.
		 */
		margin-top: 0;
	}
	.groups-has-banner .top-bar {
		background-color: transparent;
		box-shadow: none;
	}
	.groups-has-banner  windowhandle,
	.groups-has-banner .groups-header {
		transition: linear 65ms all;
	}
	.groups-has-banner.groups-scrolled windowhandle {
		background-color: transparent;
	}
	.groups-has-banner.groups-scrolled headerbar {
		background-color: @theme_bg_color;
	}
	.groups-has-banner .groups-header {
		box-shadow: 0 0 6px 0px @theme_bg_color;
	}
	.groups-has-banner:not(.groups-scrolled) .groups-header {
		/* go run ./cmd/ease-in-out-gradient/ -max 0.25 -min 0 -steps 5 */
		background: linear-gradient(to bottom,
			alpha(black, 0.24),
			alpha(black, 0.19),
			alpha(black, 0.06),
			alpha(black, 0.01),
			alpha(black, 0.00) 100%
		);
		box-shadow: none;
		border: none;
	}
	.groups-has-banner .groups-banner-shadow {
		background: alpha(black, 0.75);
	}
	.groups-has-banner:not(.groups-scrolled) .groups-header * {
		color: white;
		text-shadow: 0px 0px 5px alpha(black, 0.75);
	}
	.groups-has-banner:not(.groups-scrolled) .groups-header *:backdrop {
		color: alpha(white, 0.75);
		text-shadow: 0px 0px 2px alpha(black, 0.35);
	}
	.groups-name {
		font-weight: 600;
		font-size: 1.1em;
	}
`)

// NewView creates a new GroupsView.
func NewView(ctx context.Context, relayURL string) *GroupsView {
	v := GroupsView{
		RelayURL: relayURL,
	}

	v.ToolbarView = adw.NewToolbarView()
	v.ToolbarView.SetTopBarStyle(adw.ToolbarFlat)
	v.ToolbarView.SetExtendContentToTopEdge(true) // basically act like an overlay

	// Bind the context to cancel when we're hidden.
	v.ctx = gtkutil.WithVisibility(ctx, v)

	v.Header.Name = gtk.NewLabel("")
	v.Header.Name.AddCSSClass("groups-name")
	v.Header.Name.SetHAlign(gtk.AlignStart)
	v.Header.Name.SetEllipsize(pango.EllipsizeEnd)

	// The header is placed on top of the overlay, kind of like the official
	// client.
	v.Header.HeaderBar = adw.NewHeaderBar()
	v.Header.HeaderBar.AddCSSClass("groups-header")
	v.Header.HeaderBar.SetShowTitle(false)
	v.Header.HeaderBar.PackStart(v.Header.Name)

	viewport := gtk.NewViewport(nil, nil)

	v.Scroll = gtk.NewScrolledWindow()
	v.Scroll.AddCSSClass("groups-view-scroll")
	v.Scroll.SetVExpand(true)
	v.Scroll.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	v.Scroll.SetChild(viewport)
	// v.Scroll.SetPropagateNaturalWidth(true)
	// v.Scroll.SetPropagateNaturalHeight(true)

	var headerScrolled bool

	vadj := v.Scroll.VAdjustment()
	vadj.ConnectValueChanged(func() {
		if scrolled := v.Child.Banner.SetScrollOpacity(vadj.Value()); scrolled {
			if !headerScrolled {
				headerScrolled = true
				v.AddCSSClass("groups-scrolled")
			}
		} else {
			if headerScrolled {
				headerScrolled = false
				v.RemoveCSSClass("groups-scrolled")
			}
		}
	})

	v.Child = NewGroupsView(ctx)

	groups := make([]*Group, 0, 4)
	var lastOpen nip29.GroupAddress
	handleSelect := func(gad nip29.GroupAddress) {
		if lastOpen.Equals(gad) {
			return
		}

		app.NewStateKey[string]("last-group").Acquire(ctx).Set(gad.Relay, gad.ID)
		for _, g := range groups {
			if g.gad.Equals(gad) {
				continue
			}
			g.unselect()
		}
	}

	go func() {
		me := global.GetMe(ctx)
		for {
			select {
			case group := <-me.JoinedGroup:
				g := NewGroup(ctx, group, handleSelect)
				groups = append(groups, g)
				fmt.Println("joined", g)
				v.Child.append(g)
			case gad := <-me.LeftGroup:
				v.Child.remove(gad)
			}
		}
	}()

	viewport.SetChild(v.Child)
	viewport.SetFocusChild(v.Child)

	v.ToolbarView.AddTopBar(v.Header)
	v.ToolbarView.SetContent(v.Scroll)
	v.ToolbarView.SetFocusChild(v.Scroll)

	viewCSS(v)
	return &v
}

// InvalidateHeader invalidates the relay name and banner.
func (v *GroupsView) InvalidateHeader() {
	// state := gtkcord.FromContext(v.ctx.Take())

	// g, err := state.Cabinet.Relay(v.relayID)
	// if err != nil {
	// 	log.Printf("groups.GroupsView: cannot fetch relay %d: %v", v.relayID, err)
	// 	return
	// }

	// v.Header.Name.SetText(g.Name)
	// v.invalidateBanner()
}

func (v *GroupsView) invalidateBanner() {
	v.Child.Banner.Invalidate()

	if v.Child.Banner.HasBanner() {
		v.AddCSSClass("groups-has-banner")
	} else {
		v.RemoveCSSClass("groups-has-banner")
	}
}

type GroupsListView struct {
	*gtk.Box
	Children []*Group
	Banner   *Banner
}

func NewGroupsView(ctx context.Context) *GroupsListView {
	gv := &GroupsListView{}

	gv.Box = gtk.NewBox(gtk.OrientationVertical, 0)
	gv.Box.SetSizeRequest(bannerWidth, -1)
	gv.Box.AddCSSClass("groups-viewtree")
	gv.Box.SetHExpand(true)
	gv.Box.SetVExpand(true)

	// gv.Banner = NewBanner(ctx)
	// gv.Banner.Invalidate()
	// gv.Box.Append(gv.Banner)

	return gv
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

func gadFromListItem(item *gtk.ListItem) nip29.GroupAddress {
	return gadFromItem(item.Item())
}

func gadFromItem(item *glib.Object) nip29.GroupAddress {
	str := item.Cast().(*gtk.StringObject)

	gad, err := nip29.ParseGroupAddress(str.String())
	if err != nil {
		panic(fmt.Sprintf("gadFromListItem: failed to parse gad: %v", err))
	}

	return gad
}
