package utils

import (
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/nbd-wtf/go-nostr/nip29"
)

var (
	GroupAddressVariant = glib.NewVariantType("gad")
	RelayURLVariant     = glib.NewVariantType("url")
)

func NewGroupAddressVariant(gad nip29.GroupAddress) *glib.Variant {
	return glib.NewVariantString(gad.String())
}

func NewRelayURLVariant(url string) *glib.Variant {
	return glib.NewVariantString(url)
}
