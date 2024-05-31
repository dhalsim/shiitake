package main

import (
	"context"
	"regexp"

	"fiatjaf.com/shiitake/global"
	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gotkit/components/onlineimage"
	"github.com/diamondburned/gotkit/gtkutil"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
	"github.com/diamondburned/gotkit/gtkutil/imgutil"
)

type userBar struct {
	*gtk.Box
	avatar *onlineimage.Avatar
	name   *gtk.Label
	status *gtk.Image
	menu   *gtk.ToggleButton

	ctx context.Context
}

var userBarCSS = cssutil.Applier("user-bar", `
.user-bar-avatar {
  padding: 6px;
}
.user-bar-menu {
  margin: 0 6px;
}
`)

func newUserBar(ctx context.Context) *userBar {
	b := userBar{ctx: ctx}
	b.avatar = onlineimage.NewAvatar(ctx, imgutil.HTTPProvider, 14)
	b.avatar.AddCSSClass("user-bar-avatar")

	b.name = gtk.NewLabel("")
	b.name.AddCSSClass("user-bar-name")
	b.name.SetSelectable(true)
	b.name.SetXAlign(0)
	b.name.SetHExpand(true)
	b.name.SetWrap(false)
	b.name.SetEllipsize(pango.EllipsizeEnd)

	b.status = gtk.NewImage()
	b.status.AddCSSClass("user-bar-status")
	// b.updatePresence(nil)

	b.menu = gtk.NewToggleButton()
	b.menu.AddCSSClass("user-bar-menu")
	b.menu.SetIconName("menu-large-symbolic")
	b.menu.SetTooltipText("Main Menu")
	b.menu.SetHasFrame(false)
	b.menu.SetVAlign(gtk.AlignCenter)
	b.menu.ConnectClicked(func() {
		p := gtkutil.NewPopoverMenuCustom(b.menu, gtk.PosTop, []gtkutil.PopoverMenuItem{
			gtkutil.MenuItem("Quick Switcher", "win.quick-switcher"),
			gtkutil.MenuSeparator("User Settings"),
			gtkutil.Submenu("Set _Status", []gtkutil.PopoverMenuItem{
				gtkutil.MenuItem("_Online", "win.set-online"),
				gtkutil.MenuItem("_Idle", "win.set-idle"),
				gtkutil.MenuItem("_Do Not Disturb", "win.set-dnd"),
				gtkutil.MenuItem("In_visible", "win.set-invisible"),
			}),
			gtkutil.MenuSeparator(""),
			gtkutil.MenuItem("Preferences", "win.quick-switcher"),
			gtkutil.MenuItem("About", "app.about"),
			gtkutil.MenuItem("Logs", "app.logs"),
			gtkutil.MenuItem("Quit", "app.quit"),
		})
		p.ConnectHide(func() { b.menu.SetActive(false) })
		gtkutil.PopupFinally(p)
	})

	b.Box = gtk.NewBox(gtk.OrientationHorizontal, 0)
	b.Box.Append(b.avatar)
	b.Box.Append(b.name)
	b.Box.Append(b.status)
	b.Box.Append(b.menu)
	userBarCSS(b)

	anim := b.avatar.EnableAnimation()
	anim.ConnectMotion(b)

	go func() {
		me := global.GetMe(ctx)
		resetMetadata := func() {
			glib.IdleAdd(func() {
				// if v, _ := strconv.Atoi(me.Discriminator); v != 0 {
				// 	tag += `<span size="smaller">` + "#" + me.Discriminator + "</span>"
				// }

				b.avatar.SetInitials(me.ShortName())
				b.avatar.SetFromURL(me.Picture)
				b.name.SetMarkup(me.ShortName())
				b.name.SetTooltipMarkup(me.ShortName())
			})
		}
		resetMetadata()

		for range me.MetadataUpdated {
			resetMetadata()
		}
	}()

	return &b
}

var discriminatorRe = regexp.MustCompile(`#\d{1,4}$`)

// func (b *userBar) updatePresence(presence *discord.Presence) {
// 	if presence == nil {
// 		b.status.SetTooltipText(statusText(discord.UnknownStatus))
// 		b.status.SetFromIconName(statusIcon(discord.UnknownStatus))
// 		return
// 	}
//
// 	if presence.User.Username != "" {
// 		b.updateUser(&presence.User)
// 	}
//
// 	b.status.SetTooltipText(statusText(presence.Status))
// 	b.status.SetFromIconName(statusIcon(presence.Status))
// }

func (b *userBar) invalidatePresence() {
	// state := gtkcord.FromContext(b.ctx)
	// me, _ := state.Me()

	// presence, _ := state.PresenceStore.Presence(0, me.ID)
	// if presence != nil {
	// 	b.updatePresence(presence)
	// }
}
