package utils

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func ButtonLoading(button *gtk.Button, loadingText string) func() {
	previousLabel := button.Label()
	button.SetLabel(loadingText)
	button.SetSensitive(false)

	return func() {
		button.SetSensitive(true)
		button.SetLabel(previousLabel)
	}
}
