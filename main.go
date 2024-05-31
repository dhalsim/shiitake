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
	"github.com/diamondburned/gotkit/gtkutil/cssutil"

	"fiatjaf.com/shiitake/about"
	_ "fiatjaf.com/shiitake/icons"
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
	win         *Window
	application *app.Application
)

func main() {
	application = app.New(context.Background(), "com.fiatjaf.shiitake", "Shiitake")

	application.ConnectActivate(func() {
		ctx := application.Context()
		adw.Init()
		adaptive.Init()

		if win != nil {
			win.Present()
			return
		}

		win = NewWindow(ctx)
		win.Show()

		prefs.AsyncLoadSaved(ctx, func(err error) {
			if err != nil {
				app.Error(ctx, err)
			}
		})
	})
	application.RunMain()
}
