package main

import (
	"context"

	"fiatjaf.com/shiitake/about"
	"fiatjaf.com/shiitake/components/icon_placeholder"
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

var forceDarkTheme = prefs.NewBool(true, prefs.PropMeta{
	Name:        "Use Dark Theme",
	Description: "Whether or not to use dark mode even if your system is set to light.",
	Section:     "Theme",
})

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

	main *MainView
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
	w.main = NewMainView(w.ctx, &w)
	plc := icon_placeholder.New("chat-bubbles-empty-symbolic")

	w.Stack = gtk.NewStack()
	w.Stack.SetTransitionType(gtk.StackTransitionTypeCrossfade)
	w.Stack.AddChild(login)
	w.Stack.AddChild(w.main)
	w.Stack.AddChild(plc)
	win.SetContent(w.Stack)

	// show placeholder
	w.Stack.SetVisibleChild(plc)

	gtkutil.AddActions(&w, map[string]func(){
		"reset-view": func() {
			w.main.messagesView.switchTo(nip29.GroupAddress{})
		},
		"quick-switcher": func() {
			w.main.OpenQuickSwitcher()
		},
		"preferences": func() { prefui.ShowDialog(ctx) },
		"about":       func() { about.New(ctx).Present() },
		"logs": func() {
			viewer := logui.NewDefaultViewer(ctx)
			viewer.SetHideOnClose(false)
			viewer.SetDestroyWithParent(true)
			viewer.SetDefaultSize(850, -1)
			viewer.Show()
		},
		"quit": func() { application.Quit() },
	})

	gtkutil.AddActionShortcuts(&w, map[string]string{
		"<Ctrl>K": "win.quick-switcher",
		"<Ctrl>Q": "win.quit",
	})

	// attempt login with stored credentials
	login.TryLoginFromDriver(ctx)

	styleManager := adw.StyleManagerGetDefault()
	baseScheme := styleManager.ColorScheme()
	forceDarkTheme.Subscribe(
		func() {
			preferDark := forceDarkTheme.Value()
			if preferDark {
				styleManager.SetColorScheme(adw.ColorSchemePreferDark)
			} else {
				styleManager.SetColorScheme(baseScheme)
			}
			if styleManager.Dark() {
				w.AddCSSClass("dark")
			} else {
				w.RemoveCSSClass("dark")
			}
		},
	)

	return &w
}

func (w *Window) OpenGroup(gad nip29.GroupAddress) {
	eachChild(w.main.Sidebar.CurrentGroupsView.List, func(lbr *gtk.ListBoxRow) bool {
		if lbr.Name() == gad.String() {
			if w.main.Sidebar.CurrentGroupsView.List.SelectedRow() != lbr {
				w.main.Sidebar.CurrentGroupsView.List.SelectRow(lbr)
			}
			return true
		}
		return false
	})
	w.main.messagesView.switchTo(gad)
}

func (w *Window) OpenRelay(url string) {
	eachChild(w.main.Sidebar.RelaysView.Widget, func(lbr *gtk.ListBoxRow) bool {
		if lbr.Name() == url {
			if w.main.Sidebar.RelaysView.Widget.SelectedRow() != lbr {
				w.main.Sidebar.RelaysView.Widget.SelectRow(lbr)
			}
			return true
		}
		return false
	})

	w.main.Sidebar.openRelay(url)
	w.main.messagesView.switchTo(nip29.GroupAddress{})
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
