package relays

import (
	"context"

	"fiatjaf.com/shiitake/components/form_entry"
	"fiatjaf.com/shiitake/sidebar/sidebutton"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/app"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
)

type Button struct {
	*gtk.Overlay
	Pill   *sidebutton.Pill
	Button *gtk.Button

	ctx context.Context
}

var buttonCSS = cssutil.Applier("sidebar-add-button-overlay", `
	.sidebar-add-button {
		padding: 4px 12px;
		border-radius: 0;
	}
	.sidebar-add-button image {
		padding-top: 4px;
		padding-bottom: 2px;
	}
`)

func NewButton(ctx context.Context, done func(string)) *Button {
	b := Button{ctx: ctx}

	icon := gtk.NewImageFromIconName("list-add-symbolic")
	icon.SetIconSize(gtk.IconSizeLarge)
	icon.SetPixelSize(10)

	b.Button = gtk.NewButton()
	b.Button.AddCSSClass("sidebar-button")
	b.Button.SetTooltipText("Add New Group")
	b.Button.SetChild(icon)
	b.Button.SetHasFrame(false)
	b.Button.ConnectClicked(func() {
		b.Pill.State = sidebutton.PillActive
		b.Pill.Invalidate()

		form := form_entry.New("Relay URL or Group Address")
		form.FocusNextOnActivate()

		prompt := gtk.NewDialog()
		prompt.SetTitle("Add New Group")
		prompt.SetDefaultSize(250, 80)
		prompt.SetTransientFor(app.GTKWindowFromContext(ctx))
		prompt.SetModal(true)
		prompt.AddButton("Cancel", int(gtk.ResponseCancel))
		prompt.AddButton("OK", int(gtk.ResponseAccept))
		prompt.SetDefaultResponse(int(gtk.ResponseAccept))

		inner := prompt.ContentArea()
		inner.Append(form)
		inner.SetVExpand(true)
		inner.SetHExpand(true)
		inner.SetVAlign(gtk.AlignCenter)
		inner.SetHAlign(gtk.AlignCenter)
		// passwordCSS(passInner)

		form.Entry.ConnectActivate(func() {
			// Enter key activates.
			prompt.Response(int(gtk.ResponseAccept))
		})

		prompt.ConnectResponse(func(id int) {
			defer prompt.Close()

			switch id {
			case int(gtk.ResponseCancel):
				done("")
			case int(gtk.ResponseAccept):
				done(form.Text())
			}
		})
		prompt.Show()

		parent := gtk.BaseWidget(b.Button.Parent())
		parent.ActivateAction("win.add-new", nil)
	})

	b.Pill = sidebutton.NewPill()

	b.Overlay = gtk.NewOverlay()
	b.Overlay.SetChild(b.Button)
	b.Overlay.AddOverlay(b.Pill)

	buttonCSS(b)
	return &b
}
