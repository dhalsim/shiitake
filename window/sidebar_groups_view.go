package window

import (
	"context"
	"fmt"

	"fiatjaf.com/shiitake/global"
	"fiatjaf.com/shiitake/utils"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gotkit/app"
	"github.com/diamondburned/gotkit/gtkutil"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
	"github.com/nbd-wtf/go-nostr/nip29"
)

const groupsWidth = 400

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

var sidebarViewCSS = cssutil.Applier("groups-view", `
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
	.groups-name {
		font-weight: 600;
		font-size: 1.1em;
	}
`)

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
		if headerScrolled {
			headerScrolled = false
			v.RemoveCSSClass("groups-scrolled")
		}
	})

	v.Child = NewGroupsListView(ctx)

	groups := make([]*Group, 0, 4)
	var lastOpen nip29.GroupAddress
	handleSelect := func(gad nip29.GroupAddress) {
		if lastOpen.Equals(gad) {
			return
		}

		parent := gtk.BaseWidget(v.Parent())
		parent.ActivateAction("win.open-group", utils.NewGroupAddressVariant(gad))

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

	sidebarViewCSS(v)
	return &v
}

type GroupsListView struct {
	*gtk.Box
	Children []*Group
}

func NewGroupsListView(ctx context.Context) *GroupsListView {
	gv := &GroupsListView{}

	gv.Box = gtk.NewBox(gtk.OrientationVertical, 0)
	gv.Box.SetSizeRequest(groupsWidth, -1)
	gv.Box.AddCSSClass("groups-viewtree")
	gv.Box.SetHExpand(true)
	gv.Box.SetVExpand(true)

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
