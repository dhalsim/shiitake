package main

import (
	"context"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/nbd-wtf/go-nostr/nip29"
)

type MainView struct {
	*gtk.Box

	Sidebar  *Sidebar
	Messages *MessagesView
	Discover *DiscoverView

	Header *adw.HeaderBar
	Stack  *gtk.Stack

	ctx context.Context
}

func NewMainView(ctx context.Context, w *Window) *MainView {
	p := MainView{
		ctx: ctx,
	}

	p.Sidebar = NewSidebar(ctx)

	p.Messages = NewMessagesView(ctx)
	p.Discover = NewDiscoverView(ctx)

	p.Stack = gtk.NewStack()
	p.Stack.AddChild(p.Discover)
	p.Stack.AddChild(p.Messages)
	p.Stack.SetVisibleChild(p.Discover)

	rightTitle := gtk.NewLabel("")
	rightTitle.SetXAlign(0)
	rightTitle.SetHExpand(true)
	rightTitle.SetEllipsize(pango.EllipsizeEnd)

	p.Header = adw.NewHeaderBar()
	p.Header.SetShowEndTitleButtons(true)
	p.Header.SetShowBackButton(true)
	p.Header.SetShowTitle(true)

	paned := gtk.NewPaned(gtk.OrientationHorizontal)
	paned.SetHExpand(true)
	paned.SetVExpand(true)
	paned.SetStartChild(p.Sidebar)
	paned.SetEndChild(p.Stack)
	paned.SetPosition(160)
	paned.SetResizeStartChild(true)
	paned.SetResizeEndChild(true)
	paned.Show()

	p.Box = gtk.NewBox(gtk.OrientationVertical, 0)
	p.Box.Append(p.Header)
	p.Box.Append(paned)

	// state := gtkcord.FromContext(ctx)
	// w.ConnectDestroy(state.AddHandler(
	// 	func(*gateway.MessageCreateEvent) { p.updateWindowTitle() },
	// 	func(*gateway.MessageUpdateEvent) { p.updateWindowTitle() },
	// 	func(*gateway.MessageDeleteEvent) { p.updateWindowTitle() },
	// 	func(*read.UpdateEvent) { p.updateWindowTitle() },
	// ))

	return &p
}

func (p *MainView) OpenDiscover() {
	p.Stack.SetVisibleChild(p.Discover)
	p.Header.SetTitleWidget(adw.NewWindowTitle("Discover", ""))
}

func (p *MainView) OpenGroup(gad nip29.GroupAddress) {
	eachChild(p.Sidebar.GroupsView.List, func(lbr *gtk.ListBoxRow) bool {
		if lbr.Name() == gad.String() {
			if p.Sidebar.GroupsView.List.SelectedRow() != lbr {
				p.Sidebar.GroupsView.List.SelectRow(lbr)
			}
			return true
		}
		return false
	})
	p.Stack.SetVisibleChild(p.Messages)
	p.Messages.switchTo(gad)
}
