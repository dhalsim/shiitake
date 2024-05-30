package main

import (
	"context"
	"embed"
	"io/fs"

	"github.com/diamondburned/adaptive"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotkit/app"
	"github.com/diamondburned/gotkit/app/locale"
	"github.com/diamondburned/gotkit/app/prefs"
	"github.com/diamondburned/gotkit/components/logui"
	"github.com/diamondburned/gotkit/components/prefui"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"

	_ "fiatjaf.com/shiitake/icons"
	"fiatjaf.com/shiitake/window"
	"fiatjaf.com/shiitake/window/about"
	_ "github.com/diamondburned/gotkit/gtkutil/aggressivegc"
)

//go:embed po/*
var po embed.FS

func init() {
	po, _ := fs.Sub(po, "po")
	locale.LoadLocale(po)
}

// Version is connected to about.SetVersion.
var Version string

func init() {
	about.SetVersion(Version)
	initDimensions()
}

var _ = cssutil.WriteCSS(`
	window.background,
	window.background.solid-csd {
		background-color: @theme_bg_color;
	}

	.adaptive-avatar > image {
		background: none;
	}
	.adaptive-avatar > label {
		background: @borders;
	}
`)

var (
	win         *window.Window
	application *app.Application
)

func main() {
	application = app.New(context.Background(), "com.fiatjaf.shiitake", "Shiitake")

	application.AddJSONActions(map[string]interface{}{
		"application.preferences": func() { prefui.ShowDialog(win.Context()) },
		"application.about":       func() { about.New(win.Context()).Present() },
		"application.logs":        func() { logui.ShowDefaultViewer(win.Context()) },
		"application.quit":        func() { application.Quit() },
	})
	application.AddActionShortcuts(map[string]string{
		"<Ctrl>Q": "application.quit",
	})
	application.ConnectActivate(func() {
		ctx := application.Context()
		adw.Init()
		adaptive.Init()

		if win != nil {
			win.Present()
			return
		}

		win = window.NewWindow(ctx)
		win.Show()

		prefs.AsyncLoadSaved(ctx, func(err error) {
			if err != nil {
				app.Error(ctx, err)
			}
		})
	})
	application.RunMain()
}
