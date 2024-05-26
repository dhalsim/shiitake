package window

import (
	"context"

	"fiatjaf.com/shiitake/global"
	"fiatjaf.com/shiitake/messages"
	"fiatjaf.com/shiitake/window/backbutton"
	"fiatjaf.com/shiitake/window/quickswitcher"
	"github.com/diamondburned/adaptive"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gotkit/app"
	"github.com/diamondburned/gotkit/gtkutil"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
	"github.com/nbd-wtf/go-nostr/nip29"
)

var (
	lastRelay = app.NewSingleStateKey[string]("last-relay")
	lastGroup = app.NewStateKey[string]("last-group")
)

// TODO: refactor this to support TabOverview. We do this by refactoring Sidebar
// out completely and merging it into ChatPage. We can then get rid of the logic
// to keep the Sidebar in sync with the ChatPage, since each tab will have its
// own Sidebar.

type ChatPage struct {
	*adw.OverlaySplitView
	Sidebar     *Sidebar
	RightHeader *adw.HeaderBar
	RightTitle  *gtk.Label

	chatView      *ChatView
	quickswitcher *quickswitcher.Dialog

	lastRelay         *app.TypedSingleState[string]
	lastGroupForRelay *app.TypedState[string]

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
		ctx:               ctx,
		lastRelay:         lastRelay.Acquire(ctx),
		lastGroupForRelay: lastGroup.Acquire(ctx),
	}

	p.quickswitcher = quickswitcher.NewDialog(ctx)
	p.quickswitcher.SetHideOnClose(true) // so we can reopen it later

	p.chatView = NewChatView(ctx)

	p.Sidebar = NewSidebar(ctx)
	p.Sidebar.SetHAlign(gtk.AlignStart)

	p.RightTitle = gtk.NewLabel("")
	p.RightTitle.AddCSSClass("right-header-label")
	p.RightTitle.SetXAlign(0)
	p.RightTitle.SetHExpand(true)
	p.RightTitle.SetEllipsize(pango.EllipsizeEnd)

	back := backbutton.New()
	back.SetTransitionType(gtk.RevealerTransitionTypeSlideRight)

	joinGroupButton := gtk.NewButtonFromIconName("list-add-symbolic")
	joinGroupButton.SetTooltipText("Join Group")
	joinGroupButton.ConnectClicked(p.AskJoinGroup)

	p.RightHeader = adw.NewHeaderBar()
	p.RightHeader.AddCSSClass("right-header")
	p.RightHeader.SetShowEndTitleButtons(true)
	p.RightHeader.SetShowBackButton(false) // this is useless with OverlaySplitView
	p.RightHeader.SetShowTitle(false)
	p.RightHeader.PackStart(back)
	p.RightHeader.PackStart(p.RightTitle)
	p.RightHeader.PackEnd(joinGroupButton)

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

// ResetView switches out of any channel view and into the placeholder view.
// This method is used when the guild becomes unavailable.
func (p *ChatPage) ResetView() { p.SwitchToPlaceholder() }

// SwitchToPlaceholder switches to the empty placeholder view.
func (p *ChatPage) SwitchToPlaceholder() {
	p.chatView.switchToPlaceholder()
}

// SwitchToMessages reopens a new message page of the same channel ID if the
// user is opening one. Otherwise, the placeholder is seen.
func (p *ChatPage) SwitchToMessages() {
	p.chatView.switchToPlaceholder()

	p.lastRelay.Exists(func(exists bool) {
		if !exists {
			return
		}
		// Restore the last opened channel if there is one.
		p.lastRelay.Get(func(rl string) {
			p.lastGroupForRelay.Get(rl, func(id string) {
				p.OpenGroup(nip29.GroupAddress{Relay: rl, ID: id})
			})
		})
	})
}

// OpenRelay opens the relay with the given ID.
func (p *ChatPage) OpenRelay(relayURL string) {
	p.lastRelay.Set(relayURL)
	p.Sidebar.openRelay(relayURL)
	p.restoreLastGroup()
}

func (p *ChatPage) restoreLastGroup() {
	// Allow a bit of delay for the page to finish loading.
	glib.IdleAdd(func() {
		p.lastRelay.Exists(func(exists bool) {
			if exists {
				p.lastRelay.Get(func(rl string) {
					p.lastGroupForRelay.Get(rl, func(id string) {
						p.OpenGroup(nip29.GroupAddress{Relay: rl, ID: id})
					})
				})
			} else {
				p.SwitchToPlaceholder()
			}
		})
	})
}

// OpenGroup opens the group with the given ID. Use this method to direct
// the user to a new channel when they request to, e.g. through a notification.
func (p *ChatPage) OpenGroup(gad nip29.GroupAddress) {
	p.chatView.switchToGroup(gad)

	group := global.GetGroup(p.ctx, gad)
	if group != nil {
		// Save the last opened channel for the guild.
		p.lastGroupForRelay.Set(gad.Relay, gad.ID)
	}
}

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
	messageView *messages.MessagesView // nilable
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

	return &t
}

func (t *ChatView) Current() nip29.GroupAddress {
	if t.messageView == nil {
		return nip29.GroupAddress{}
	}
	return t.messageView.Group.Address
}

func (t *ChatView) switchToPlaceholder() bool {
	return t.switchToGroup(nip29.GroupAddress{})
}

func (t *ChatView) switchToGroup(gad nip29.GroupAddress) bool {
	if t.Current().Equals(gad) {
		return false
	}

	old := t.messageView

	t.messageView = messages.NewMessagesView(t.ctx, gad)

	t.Stack.AddChild(t.messageView)
	t.Stack.SetVisibleChild(t.messageView)

	viewWidget := gtk.BaseWidget(t.messageView)
	viewWidget.GrabFocus()

	if old != nil {
		gtkutil.NotifyProperty(t.Stack, "transition-running", func() bool {
			if !t.Stack.TransitionRunning() {
				t.Stack.Remove(old)
				return true
			}
			return false
		})
	}

	return true
}

func newEmptyMessagePlaceholder() gtk.Widgetter {
	status := adaptive.NewStatusPage()
	status.SetIconName("chat-bubbles-empty-symbolic")
	status.Icon.SetOpacity(0.45)
	status.Icon.SetIconSize(gtk.IconSizeLarge)

	return status
}
