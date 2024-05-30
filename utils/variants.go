package utils

import (
	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

var EventIDVariant = glib.NewVariantType("s")

func NewEventIDVariant(id string) *glib.Variant {
	return glib.NewVariantString(id)
}
