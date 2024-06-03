package main

import (
	"context"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type AddRelayButton struct {
	*gtk.Button
	ctx context.Context
}

func NewAddRelayButton(ctx context.Context, done func(string)) *AddRelayButton {
	b := AddRelayButton{ctx: ctx}

	icon := gtk.NewImageFromIconName("list-add-symbolic")
	icon.SetIconSize(gtk.IconSizeLarge)
	icon.SetPixelSize(10)

	b.Button = gtk.NewButton()
	b.Button.SetTooltipText("Add New Group")
	b.Button.SetChild(icon)
	b.Button.SetHasFrame(false)
	b.Button.ConnectClicked(func() {
	})

	return &b
}

type RelayView struct {
	*adw.PreferencesPage
	ctx context.Context
}

func NewRelayView(ctx context.Context) *RelayView {
	r := RelayView{
		ctx: ctx,
	}

	r.PreferencesPage = adw.NewPreferencesPage()

	return &r
}
