package main

import (
	"context"

	"fiatjaf.com/shiitake/about"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/app"
	"github.com/diamondburned/gotkit/app/prefs"
	"github.com/diamondburned/gotkit/components/logui"
	"github.com/diamondburned/gotkit/components/prefui"
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

	chat *ChatPage
}

func NewWindow(ctx context.Context) *Window {
	application := app.FromContext(ctx)

	win := adw.NewApplicationWindow(application.Application)
	win.SetSizeRequest(320, 320)
	win.SetDefaultSize(800, 600)

	appWindow := app.WrapWindow(application, &win.ApplicationWindow)
	appWindow.SetResizable(true)
	appWindow.SetTitle("shiitake")
	ctx = app.WithWindow(ctx, appWindow)

	w := Window{
		ApplicationWindow: win,
		win:               appWindow,
		ctx:               ctx,
	}
	w.ctx = ctxt.With(w.ctx, &w)

	login := NewLoginPage(ctx, &w)
	w.chat = NewChatPage(w.ctx, &w)
	plc := newEmptyMessagePlaceholder()

	w.Stack = gtk.NewStack()
	w.Stack.SetTransitionType(gtk.StackTransitionTypeCrossfade)
	w.Stack.AddChild(login)
	w.Stack.AddChild(w.chat)
	w.Stack.AddChild(plc)
	win.SetContent(w.Stack)

	// show placeholder
	w.Stack.SetVisibleChild(plc)

	gtkutil.AddActions(&w, map[string]func(){
		"reset-view": func() {
			w.chat.chatView.switchToGroup(nip29.GroupAddress{})
		},
		"quick-switcher": func() {
			w.chat.OpenQuickSwitcher()
		},
		"preferences": func() { prefui.ShowDialog(ctx) },
		"about":       func() { about.New(ctx).Present() },
		"logs":        func() { logui.ShowDefaultViewer(ctx) },
		"quit":        func() { application.Quit() },
	})

	gtkutil.AddActionShortcuts(&w, map[string]string{
		"<Ctrl>K": "win.quick-switcher",
		"<Ctrl>Q": "win.quit",
	})

	// attempt login with stored credentials
	login.TryLoginFromDriver()

	return &w
}

func (w *Window) OpenGroup(gad nip29.GroupAddress) {
	eachChild(w.chat.Sidebar.CurrentGroupsView.List, func(lbr *gtk.ListBoxRow) bool {
		if lbr.Name() == gad.String() {
			if w.chat.Sidebar.CurrentGroupsView.List.SelectedRow() != lbr {
				w.chat.Sidebar.CurrentGroupsView.List.SelectRow(lbr)
			}
			return true
		}
		return false
	})
	w.chat.chatView.switchToGroup(gad)
}

func (w *Window) OpenRelay(url string) {
	eachChild(w.chat.Sidebar.RelaysView.Widget, func(lbr *gtk.ListBoxRow) bool {
		if lbr.Name() == url {
			if w.chat.Sidebar.RelaysView.Widget.SelectedRow() != lbr {
				w.chat.Sidebar.RelaysView.Widget.SelectRow(lbr)
			}
			return true
		}
		return false
	})

	w.chat.Sidebar.openRelay(url)
	w.chat.chatView.switchToGroup(nip29.GroupAddress{})
}

func (w *Window) SetTitle(title string) {
	w.ApplicationWindow.SetTitle(app.FromContext(w.ctx).SuffixedTitle(title))
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
