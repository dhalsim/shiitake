package directbutton

import (
	"context"

	"fiatjaf.com/shiitake/sidebar/sidebutton"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
)

type Button struct {
	*gtk.Overlay
	Pill   *sidebutton.Pill
	Button *gtk.Button

	ctx context.Context
}

var dmButtonCSS = cssutil.Applier("sidebar-dm-button-overlay", `
	.sidebar-dm-button {
		padding: 4px 12px;
		border-radius: 0;
	}
	.sidebar-dm-button image {
		padding-top: 4px;
		padding-bottom: 2px;
	}
`)

func NewButton(ctx context.Context) *Button {
	b := Button{ctx: ctx}

	icon := gtk.NewImageFromIconName("chat-bubbles-empty-symbolic")
	icon.SetIconSize(gtk.IconSizeLarge)
	icon.SetPixelSize(10)

	b.Button = gtk.NewButton()
	b.Button.AddCSSClass("sidebar-dm-button")
	b.Button.SetTooltipText("Direct Messages")
	b.Button.SetChild(icon)
	b.Button.SetHasFrame(false)
	b.Button.ConnectClicked(func() {
		b.Pill.State = sidebutton.PillActive
		b.Pill.Invalidate()

		parent := gtk.BaseWidget(b.Button.Parent())
		parent.ActivateAction("win.open-dms", nil)
	})

	b.Pill = sidebutton.NewPill()

	b.Overlay = gtk.NewOverlay()
	b.Overlay.SetChild(b.Button)
	b.Overlay.AddOverlay(b.Pill)

	dmButtonCSS(b)
	return &b
}
