package main

import (
	"context"

	"fiatjaf.com/shiitake/global"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gotkit/app"
	"github.com/nbd-wtf/go-nostr/nip29"
)

type MainView struct {
	*gtk.Box

	Sidebar  *Sidebar
	Messages *MessagesView
	Discover *DiscoverView

	Stack *gtk.Stack

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

	joinGroupButton := gtk.NewButtonFromIconName("list-add-symbolic")
	joinGroupButton.SetTooltipText("Join Group")
	joinGroupButton.ConnectClicked(p.AskJoinGroup)

	header := adw.NewHeaderBar()
	header.SetShowEndTitleButtons(true)
	header.SetShowBackButton(false)
	header.SetShowTitle(false)

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
	p.Box.Append(header)
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

func (p *MainView) AskJoinGroup() {
	entry := gtk.NewEntry()
	entry.SetInputPurpose(gtk.InputPurposeFreeForm)
	entry.SetVisibility(false)

	label := gtk.NewLabel("Enter group address:")
	label.SetXAlign(0)

	box := gtk.NewBox(gtk.OrientationVertical, 0)
	box.Append(label)
	box.Append(entry)

	prompt := gtk.NewDialog()
	prompt.SetTitle("File")
	prompt.SetDefaultSize(250, 80)
	prompt.SetTransientFor(app.GTKWindowFromContext(p.ctx))
	prompt.SetModal(true)
	prompt.AddButton("Cancel", int(gtk.ResponseCancel))
	prompt.AddButton("OK", int(gtk.ResponseAccept))
	prompt.SetDefaultResponse(int(gtk.ResponseAccept))

	inner := prompt.ContentArea()
	inner.Append(box)
	inner.SetVExpand(true)
	inner.SetHExpand(true)
	inner.SetVAlign(gtk.AlignCenter)
	inner.SetHAlign(gtk.AlignCenter)

	entry.ConnectActivate(func() {
		// Enter key activates.
		prompt.Response(int(gtk.ResponseAccept))
	})

	prompt.ConnectResponse(func(id int) {
		defer prompt.Close()
		gad, err := nip29.ParseGroupAddress(entry.Text())
		if err != nil {
			return
		}

		switch id {
		case int(gtk.ResponseAccept):
			global.JoinGroup(p.ctx, gad)
		}
	})

	prompt.Show()
}

func (p *MainView) updateWindowTitle() {
	// state := gtkcord.FromContext(p.ctx)

	// // Add a ping indicator if the user has pings.
	// mentions := state.ReadState.TotalMentionCount()
	// if mentions > 0 {
	// 	title = fmt.Sprintf("(%d) %s", mentions, title)
	// }

	// win, _ := ctxt.From[*Window](p.ctx)
	// win.SetTitle(title)
}
