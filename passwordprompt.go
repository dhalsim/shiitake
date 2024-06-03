package main

import (
	"context"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gotkit/app"
	"github.com/diamondburned/gotkit/gtkutil/textutil"
)

var inputLabelAttrs = textutil.Attrs(
	pango.NewAttrForegroundAlpha(65535 * 90 / 100), // 90%
)

// promptPassword opens a password prompt modal.
func promptPassword(ctx context.Context, done func(ok bool, password string)) {
	passEntry := gtk.NewEntry()
	passEntry.SetInputPurpose(gtk.InputPurposePassword)
	passEntry.SetVisibility(false)

	passLabel := gtk.NewLabel("Enter password:")
	passLabel.SetAttributes(inputLabelAttrs)
	passLabel.SetXAlign(0)

	passBox := gtk.NewBox(gtk.OrientationVertical, 0)
	passBox.Append(passLabel)
	passBox.Append(passEntry)

	passPrompt := gtk.NewDialog()
	passPrompt.SetTitle("File")
	passPrompt.SetDefaultSize(250, 80)
	passPrompt.SetTransientFor(app.GTKWindowFromContext(ctx))
	passPrompt.SetModal(true)
	passPrompt.AddButton("Cancel", int(gtk.ResponseCancel))
	passPrompt.AddButton("OK", int(gtk.ResponseAccept))
	passPrompt.SetDefaultResponse(int(gtk.ResponseAccept))

	passInner := passPrompt.ContentArea()
	passInner.Append(passBox)
	passInner.SetVExpand(true)
	passInner.SetHExpand(true)
	passInner.SetVAlign(gtk.AlignCenter)
	passInner.SetHAlign(gtk.AlignCenter)

	passEntry.ConnectActivate(func() {
		// Enter key activates.
		passPrompt.Response(int(gtk.ResponseAccept))
	})

	passPrompt.ConnectResponse(func(id int) {
		defer passPrompt.Close()

		password := passEntry.Text()

		switch id {
		case int(gtk.ResponseCancel):
			done(false, "")

		case int(gtk.ResponseAccept):
			done(true, password)
		}
	})

	passPrompt.Show()
}
