package main

import (
	"context"
	"embed"
	"io/fs"

	"github.com/diamondburned/adaptive"
	"github.com/diamondburned/chatkit/md/hl"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotkit/app"
	"github.com/diamondburned/gotkit/app/locale"
	"github.com/diamondburned/gotkit/app/prefs"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
	"github.com/diamondburned/gotkit/gtkutil/textutil"

	"fiatjaf.com/shiitake/about"
	_ "fiatjaf.com/shiitake/icons"
	_ "github.com/diamondburned/gotkit/gtkutil/aggressivegc"
)

//go:embed bundle.css
var css string

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
}

var (
	win         *Window
	application *app.Application
)

func main() {
	cssutil.WriteCSS(css)

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
				return
			}

			// choose values and hide extraneous options from libraries from our menu
			prefs.Hide(hl.Style)
			hl.Style.Publish("nord")
			prefs.Hide(textutil.TabWidth)
			textutil.TabWidth.Publish(2)
		})
	})

	// run gtk application
	application.RunMain()
}
