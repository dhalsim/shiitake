package icon_placeholder

import (
	"github.com/diamondburned/adaptive"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func New(icon string) gtk.Widgetter {
	status := adaptive.NewStatusPage()
	status.SetIconName(icon)
	status.Icon.SetOpacity(0.45)
	status.Icon.SetIconSize(gtk.IconSizeLarge)

	return status
}
