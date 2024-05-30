package main

import (
	"context"

	"fiatjaf.com/shiitake/components/backbutton"
	"fiatjaf.com/shiitake/global"
	"github.com/diamondburned/adaptive"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gotkit/app"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
	"github.com/nbd-wtf/go-nostr/nip29"
)

type ChatPage struct {
	*adw.OverlaySplitView
	Sidebar *Sidebar

	chatView      *ChatView
	quickswitcher *QuickSwitcherDialog

	// lastButtons keeps tracks of the header buttons of the previous view.
	// On view change, these buttons will be removed.
	lastButtons []gtk.Widgetter

	ctx context.Context
}

type chatPageView struct {
	body          gtk.Widgetter
	headerButtons []gtk.Widgetter
}

var chatPageCSS = cssutil.Applier("window-chatpage", `
	.right-header {
		border-radius: 0;
		box-shadow: none;
	}
	.right-header .adaptive-sidebar-reveal-button {
		margin: 0;
	}
	.right-header .adaptive-sidebar-reveal-button button {
		margin-left: 8px;
		margin-right: 4px;
	}
	.right-header-label {
		font-weight: bold;
	}
`)

func NewChatPage(ctx context.Context, w *Window) *ChatPage {
	p := ChatPage{
		ctx: ctx,
	}

	p.quickswitcher = NewQuickSwitcherDialog(ctx)
	p.quickswitcher.SetHideOnClose(true) // so we can reopen it later

	p.chatView = NewChatView(ctx)

	p.Sidebar = NewSidebar(ctx)
	p.Sidebar.SetHAlign(gtk.AlignStart)

	rightTitle := gtk.NewLabel("")
	rightTitle.AddCSSClass("right-header-label")
	rightTitle.SetXAlign(0)
	rightTitle.SetHExpand(true)
	rightTitle.SetEllipsize(pango.EllipsizeEnd)

	back := backbutton.New()
	back.SetTransitionType(gtk.RevealerTransitionTypeSlideRight)

	joinGroupButton := gtk.NewButtonFromIconName("list-add-symbolic")
	joinGroupButton.SetTooltipText("Join Group")
	joinGroupButton.ConnectClicked(p.AskJoinGroup)

	rightHeader := adw.NewHeaderBar()
	rightHeader.AddCSSClass("right-header")
	rightHeader.SetShowEndTitleButtons(true)
	rightHeader.SetShowBackButton(false) // this is useless with OverlaySplitView
	rightHeader.SetShowTitle(false)
	rightHeader.PackStart(back)
	rightHeader.PackStart(rightTitle)
	rightHeader.PackEnd(joinGroupButton)

	p.OverlaySplitView = adw.NewOverlaySplitView()
	p.OverlaySplitView.SetSidebar(p.Sidebar)
	p.OverlaySplitView.SetSidebarPosition(gtk.PackStart)
	p.OverlaySplitView.SetContent(p.chatView)
	p.OverlaySplitView.SetEnableHideGesture(true)
	p.OverlaySplitView.SetEnableShowGesture(true)
	p.OverlaySplitView.SetMinSidebarWidth(200)
	p.OverlaySplitView.SetMaxSidebarWidth(300)
	p.OverlaySplitView.SetSidebarWidthFraction(0.5)

	back.ConnectSplitView(p.OverlaySplitView)

	breakpoint := adw.NewBreakpoint(adw.BreakpointConditionParse("max-width: 500sp"))
	breakpoint.AddSetter(p.OverlaySplitView, "collapsed", true)
	w.AddBreakpoint(breakpoint)

	// state := gtkcord.FromContext(ctx)
	// w.ConnectDestroy(state.AddHandler(
	// 	func(*gateway.MessageCreateEvent) { p.updateWindowTitle() },
	// 	func(*gateway.MessageUpdateEvent) { p.updateWindowTitle() },
	// 	func(*gateway.MessageDeleteEvent) { p.updateWindowTitle() },
	// 	func(*read.UpdateEvent) { p.updateWindowTitle() },
	// ))

	chatPageCSS(p)
	return &p
}

func (p *ChatPage) AskJoinGroup() {
	entry := gtk.NewEntry()
	entry.SetInputPurpose(gtk.InputPurposeFreeForm)
	entry.SetVisibility(false)

	label := gtk.NewLabel("Enter group address:")
	label.SetAttributes(inputLabelAttrs)
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
	// wordCSS(inner)

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

// OpenQuickSwitcher opens the Quick Switcher dialog.
func (p *ChatPage) OpenQuickSwitcher() { p.quickswitcher.Show() }

func (p *ChatPage) updateWindowTitle() {
	// state := gtkcord.FromContext(p.ctx)

	// // Add a ping indicator if the user has pings.
	// mentions := state.ReadState.TotalMentionCount()
	// if mentions > 0 {
	// 	title = fmt.Sprintf("(%d) %s", mentions, title)
	// }

	// win, _ := ctxt.From[*Window](p.ctx)
	// win.SetTitle(title)
}

type ChatView struct {
	*gtk.Stack
	placeholder gtk.Widgetter
	messageView *MessagesView
	ctx         context.Context
}

func NewChatView(ctx context.Context) *ChatView {
	var t ChatView
	t.ctx = ctx
	t.placeholder = newEmptyMessagePlaceholder()

	t.Stack = gtk.NewStack()
	t.Stack.AddCSSClass("window-message-page")
	t.Stack.SetTransitionType(gtk.StackTransitionTypeCrossfade)
	t.Stack.AddChild(t.placeholder)
	t.Stack.SetVisibleChild(t.placeholder)

	t.messageView = NewMessagesView(ctx)
	t.Stack.AddChild(t.messageView)
	t.Stack.SetVisibleChild(t.messageView)

	return &t
}

func (t *ChatView) Current() nip29.GroupAddress {
	if t.messageView.currentGroup == nil {
		return nip29.GroupAddress{}
	}
	return t.messageView.currentGroup.Address
}

func (t *ChatView) switchToGroup(gad nip29.GroupAddress) bool {
	if t.Current().Equals(gad) {
		return false
	}

	t.messageView.switchTo(gad)

	gtk.BaseWidget(t.messageView).GrabFocus()

	return true
}

func newEmptyMessagePlaceholder() gtk.Widgetter {
	status := adaptive.NewStatusPage()
	status.SetIconName("chat-bubbles-empty-symbolic")
	status.Icon.SetOpacity(0.45)
	status.Icon.SetIconSize(gtk.IconSizeLarge)

	return status
}
