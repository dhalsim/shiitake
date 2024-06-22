package main

import (
	"context"

	"fiatjaf.com/nostr-gtk/components/avatar"
	"fiatjaf.com/shiitake/global"
	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gotkit/gtkutil"
)

type UserBar struct {
	*gtk.Box

	ctx context.Context
}

func NewUserBar(ctx context.Context) *UserBar {
	b := UserBar{ctx: ctx}

	avatar := avatar.New(ctx, 30, "")
	avatar.AddCSSClass("ml-2")

	name := gtk.NewLabel("")
	name.SetSelectable(true)
	name.SetHExpand(true)
	name.SetWrap(false)
	name.SetEllipsize(pango.EllipsizeEnd)
	name.AddCSSClass("mx-2")

	menu := gtk.NewToggleButton()
	menu.SetIconName("menu-large-symbolic")
	menu.SetTooltipText("Main Menu")
	menu.SetHasFrame(false)
	menu.SetVAlign(gtk.AlignCenter)
	menu.ConnectClicked(func() {
		p := gtkutil.NewPopoverMenuCustom(menu, gtk.PosTop, []gtkutil.PopoverMenuItem{
			gtkutil.MenuItem("Preferences", "win.preferences"),
			gtkutil.MenuItem("About", "win.about"),
			gtkutil.MenuItem("Logs", "win.logs"),
			gtkutil.MenuItem("Quit", "win.quit"),
		})
		p.ConnectHide(func() { menu.SetActive(false) })
		gtkutil.PopupFinally(p)
	})

	b.Box = gtk.NewBox(gtk.OrientationHorizontal, 0)
	b.Box.AddCSSClass("mb-2")
	b.Box.SetName("userbar")
	b.Box.SetHAlign(gtk.AlignFill)
	b.Box.Append(avatar)
	b.Box.Append(name)
	b.Box.Append(menu)

	// anim := b.avatar.EnableAnimation()
	// anim.ConnectMotion(b)

	go func() {
		me := global.GetMe(ctx)

		glib.IdleAdd(func() {
			avatar.SetText(me.PubKey)
			avatar.SetShowInitials(true)
		})

		resetMetadata := func() {
			glib.IdleAdd(func() {
				// if v, _ := strconv.Atoi(me.Discriminator); v != 0 {
				// 	tag += `<span size="smaller">` + "#" + me.Discriminator + "</span>"
				// }

				avatar.SetFromURL(me.Picture)
				name.SetMarkup(me.ShortName())
				name.SetTooltipMarkup(me.ShortName())
			})
		}
		resetMetadata()

		for range me.MetadataUpdated {
			resetMetadata()
		}
	}()

	return &b
}
