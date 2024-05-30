package main

import (
	"context"
	"sync"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/app"
	"github.com/diamondburned/gotkit/app/prefs"
	"github.com/diamondburned/gotkit/gtkutil"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
	"github.com/nbd-wtf/go-nostr/nip29"
	"libdb.so/ctxt"
)

var useDiscordColorScheme = prefs.NewBool(true, prefs.PropMeta{
	Name:        "Use Discord's color preference",
	Section:     "Discord",
	Description: "Whether or not to use Discord's dark/light mode preference.",
})

// SetPreferDarkTheme sets whether or not GTK should use a dark theme.
func SetPreferDarkTheme(prefer bool) {
	if !useDiscordColorScheme.Value() {
		return
	}

	scheme := adw.ColorSchemePreferLight
	if prefer {
		scheme = adw.ColorSchemePreferDark
	}

	adwStyles := adw.StyleManagerGetDefault()
	adwStyles.SetColorScheme(scheme)
}

var _ = cssutil.WriteCSS(`
	.titlebar {
		background-color: @headerbar_bg_color;
	}

	window.devel .titlebar {
		background-image: cross-fade(
			5% -gtk-recolor(url("resource:/org/gnome/Adwaita/styles/assets/devel-symbolic.svg")),
			image(transparent));
		background-repeat: repeat-x;
	}
`)

// Window is the main gtkcord window.
type Window struct {
	*adw.ApplicationWindow
	win *app.Window
	ctx context.Context

	Stack *gtk.Stack
	Login *LoginPage
	Chat  *ChatPage

	readyOnce sync.Once
}

func NewWindow(ctx context.Context) *Window {
	appInstance := app.FromContext(ctx)

	win := adw.NewApplicationWindow(appInstance.Application)
	win.SetSizeRequest(320, 320)
	win.SetDefaultSize(800, 600)

	appWindow := app.WrapWindow(appInstance, &win.ApplicationWindow)
	ctx = app.WithWindow(ctx, appWindow)

	w := Window{
		ApplicationWindow: win,
		win:               appWindow,
		ctx:               ctx,
	}
	w.ctx = ctxt.With(w.ctx, &w)

	w.Login = NewLoginPage(ctx, &w)

	w.Stack = gtk.NewStack()
	w.Stack.SetTransitionType(gtk.StackTransitionTypeCrossfade)
	w.Stack.AddChild(w.Login)
	w.Stack.SetVisibleChild(w.Login)
	win.SetContent(w.Stack)

	w.SwitchToLoginPage()
	return &w
}

func (w *Window) Context() context.Context {
	return w.ctx
}

func (w *Window) OnLogin() {
	w.readyOnce.Do(func() {
		w.initChatPage()
		w.initActions()
	})
	w.SwitchToChatPage()
}

func (w *Window) initChatPage() {
	w.Chat = NewChatPage(w.ctx, w)
	w.Stack.AddChild(w.Chat)
}

// It's not happy with how this requires a check for ChatPage, but it makes
// sense why these actions are bounded to Window and not ChatPage. Maybe?
// This requires long and hard thinking, which is simply too much for its
// brain.
func (w *Window) initActions() {
	gtkutil.AddActions(w, map[string]func(){
		// "set-online":     func() { w.setStatus(discord.OnlineStatus) },
		// "set-idle":       func() { w.setStatus(discord.IdleStatus) },
		// "set-dnd":        func() { w.setStatus(discord.DoNotDisturbStatus) },
		// "set-invisible":  func() { w.setStatus(discord.InvisibleStatus) },
		"reset-view": func() {
			w.useChatPage(
				func(cp *ChatPage) {
					cp.chatView.switchToGroup(nip29.GroupAddress{})
				})
		},
		"quick-switcher": func() { w.useChatPage((*ChatPage).OpenQuickSwitcher) },
	})

	gtkutil.AddActionShortcuts(w, map[string]string{
		"<Ctrl>K": "win.quick-switcher",
	})
}

func (w *Window) OpenGroup(gad nip29.GroupAddress) {
	w.useChatPage(func(p *ChatPage) {
		eachChild(p.Sidebar.Groups.List, func(lbr *gtk.ListBoxRow) bool {
			if lbr.Name() == gad.String() {
				p.Sidebar.Groups.List.SelectRow(lbr)
				return true
			}
			return false
		})
		p.chatView.switchToGroup(gad)
	})
}

func (w *Window) OpenRelay(url string) {
	w.useChatPage(func(p *ChatPage) {
		eachChild(p.Sidebar.Relays.Widget, func(lbr *gtk.ListBoxRow) bool {
			if lbr.Name() == url {
				p.Sidebar.Relays.Widget.SelectRow(lbr)
				return true
			}
			return false
		})

		p.Sidebar.openRelay(url)
		p.chatView.switchToGroup(nip29.GroupAddress{})
	})
}

func (w *Window) SwitchToChatPage() {
	w.Stack.SetVisibleChild(w.Chat)
	w.Chat.chatView.switchToGroup(nip29.GroupAddress{})
	w.SetTitle("")
}

func (w *Window) SwitchToLoginPage() {
	w.Stack.SetVisibleChild(w.Login)
	w.SetTitle("Login")
}

// SetTitle sets the window title.
func (w *Window) SetTitle(title string) {
	w.ApplicationWindow.SetTitle(app.FromContext(w.ctx).SuffixedTitle(title))
}

func (w *Window) showQuickSwitcher() {
	w.useChatPage(func(*ChatPage) {
		ShowQuickSwitcherDialog(w.ctx)
	})
}

func (w *Window) useChatPage(f func(*ChatPage)) {
	if w.Chat != nil {
		f(w.Chat)
	}
}

// func (w *Window) setStatus(status discord.Status) {
// 	w.useChatPage(func(*ChatPage) {
// 		state := gtkcord.FromContext(w.ctx).Online()
// 		go func() {
// 			if err := state.SetStatus(status, nil); err != nil {
// 				app.Error(w.ctx, errors.Wrap(err, "invalid status"))
// 			}
// 		}()
// 	})
// }

var emptyHeaderCSS = cssutil.Applier("empty-header", `
	.empty-header {
		min-height: 0;
		min-width: 0;
		padding: 0;
		margin: 0;
		border: 0;
	}
`)

func newEmptyHeader() *gtk.Box {
	b := gtk.NewBox(gtk.OrientationVertical, 0)
	emptyHeaderCSS(b)
	return b
}
