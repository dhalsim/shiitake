package groups

import (
	"context"
	"log"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gotkit/gtkutil"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
	"github.com/nbd-wtf/go-nostr/nip29"
)

// Refactor notice
//
// We should probably settle for an API that's kind of like this:
//
//    ch := NewView(ctx, ctrl, relayID)
//    var signal glib.SignalHandle
//    signal = ch.ConnectOnUpdate(func() bool {
//        if node := ch.Node(wantedChID); node != nil {
//            node.Select()
//            ch.HandlerDisconnect(signal)
//        }
//    })
//    ch.Invalidate()
//

const GroupsWidth = bannerWidth

// View holds the entire group sidebar containing all the categories, groups
// and threads.
type View struct {
	*adw.ToolbarView

	Header struct {
		*adw.HeaderBar
		Name *gtk.Label
	}

	Scroll *gtk.ScrolledWindow
	Child  struct {
		*gtk.Box
		Banner *Banner
		View   *gtk.ListView
	}

	ctx gtkutil.Cancellable

	model     *groupsModelManager
	selection *gtk.SingleSelection

	relayURL  string
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

// NewView creates a new View.
func NewView(ctx context.Context, relayURL string) *View {
	// state.MemberState.Subscribe(relayID)

	v := View{
		model:    newGroupsModelManager(relayURL),
		relayURL: relayURL,
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

	v.Child.Banner = NewBanner(ctx, relayURL)
	v.Child.Banner.Invalidate()

	v.selection = gtk.NewSingleSelection(v.model)
	v.selection.SetAutoselect(false)
	v.selection.SetCanUnselect(true)

	v.Child.View = gtk.NewListView(v.selection, newGroupItemFactory(ctx, v.model))
	v.Child.View.SetSizeRequest(bannerWidth, -1)
	v.Child.View.AddCSSClass("groups-viewtree")
	v.Child.View.SetVExpand(true)
	v.Child.View.SetHExpand(true)

	v.Child.Box = gtk.NewBox(gtk.OrientationVertical, 0)
	v.Child.Box.SetVExpand(true)
	v.Child.Box.Append(v.Child.Banner)
	v.Child.Box.Append(v.Child.View)
	v.Child.Box.SetFocusChild(v.Child.View)

	viewport.SetChild(v.Child)
	viewport.SetFocusChild(v.Child)

	v.ToolbarView.AddTopBar(v.Header)
	v.ToolbarView.SetContent(v.Scroll)
	v.ToolbarView.SetFocusChild(v.Scroll)

	var lastOpen nip29.GroupAddress

	v.selection.ConnectSelectionChanged(func(position, nItems uint) {
		item := v.selection.SelectedItem()
		if item == nil {
			// ctrl.OpenGroup(0)
			return
		}

		gad := gadFromItem(item)

		if lastOpen.Equals(gad) {
			return
		}
		lastOpen = gad

		// ch, _ := state.Cabinet.Group(gad)
		// if ch == nil {
		// 	log.Printf("groups.View: tried opening non-existent group %d", gad)
		// 	return
		// }

		// switch ch.Type {
		// case discord.RelayCategory, discord.RelayForum:
		// 	// We cannot display these group types.
		// 	// TODO: implement forum browsing
		// 	log.Printf("groups.View: ignoring group %d of type %d", gad, ch.Type)
		// 	return
		// }

		// log.Printf("groups.View: selected group %d", gad)

		// v.selectGAD = 0

		// row := v.model.Row(v.selection.Selected())
		// row.SetExpanded(true)

		// parent := gtk.BaseWidget(v.Child.View.Parent())
		// parent.ActivateAction("win.open-group", gtkcord.NewGroupIDVariant(gad))
	})

	// Bind to a signal that selects any group that we need to be selected.
	// This lets the group be lazy-loaded.
	v.selection.ConnectAfter("items-changed", func() {
		if v.selectGAD.Relay == "" {
			return
		}

		log.Println("groups.View: selecting group", v.selectGAD, "after items changed")

		i, ok := v.findGroupItem(v.selectGAD)
		if ok {
			v.selection.SelectItem(i, true)
			v.selectGAD.Relay = ""
			v.selectGAD.ID = ""
		}
	})

	viewCSS(v)
	return &v
}

// SelectGroup selects a known group. If none is known, then it is selected
// later when the list is changed or never selected if the user selects
// something else.
func (v *View) SelectGroup(selectGAD nip29.GroupAddress) bool {
	i, ok := v.findGroupItem(selectGAD)
	if ok && v.selection.SelectItem(i, true) {
		log.Println("groups.View: selected group", selectGAD, "immediately at", i)
		v.selectGAD.Relay = ""
		v.selectGAD.ID = ""
		return true
	}

	log.Println("groups.View: group", selectGAD, "not found, selecting later")
	v.selectGAD = selectGAD
	return false
}

// findGroupItem finds the group item by ID.
// BUG: this function is not able to find groups within collapsed categories.
func (v *View) findGroupItem(gad nip29.GroupAddress) (uint, bool) {
	n := v.selection.NItems()
	for i := uint(0); i < n; i++ {
		item := v.selection.Item(i)
		gadf := gadFromItem(item)
		if gadf.Equals(gad) {
			return i, true
		}
	}
	// TODO: recursively search v.model so we can find collapsed groups.
	return n, false
}

// RelayID returns the view's relay ID.
func (v *View) RelayURL() string {
	return v.relayURL
}

// InvalidateHeader invalidates the relay name and banner.
func (v *View) InvalidateHeader() {
	// state := gtkcord.FromContext(v.ctx.Take())

	// g, err := state.Cabinet.Relay(v.relayID)
	// if err != nil {
	// 	log.Printf("groups.View: cannot fetch relay %d: %v", v.relayID, err)
	// 	return
	// }

	// v.Header.Name.SetText(g.Name)
	// v.invalidateBanner()
}

func (v *View) invalidateBanner() {
	v.Child.Banner.Invalidate()

	if v.Child.Banner.HasBanner() {
		v.AddCSSClass("groups-has-banner")
	} else {
		v.RemoveCSSClass("groups-has-banner")
	}
}
