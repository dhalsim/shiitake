package icon_placeholder

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func New(iconName string) gtk.Widgetter {
	icon := gtk.NewImage()
	icon.Hide()
	icon.SetIconSize(gtk.IconSizeLarge)
	icon.SetFromIconName(iconName)
	icon.SetOpacity(0.45)
	icon.SetIconSize(gtk.IconSizeLarge)

	return icon
}
