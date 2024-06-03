package sidebutton

import (
	"context"

	"fiatjaf.com/nostr-gtk/components/avatar"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/ningen/v3"
)

// Button is a widget showing a single guild icon.
type Button struct {
	*gtk.Overlay
	Button *gtk.Button

	IconOverlay *gtk.Overlay
	Icon        *avatar.Avatar
	Mentions    *MentionsIndicator

	ctx       context.Context
	mentions  int
	indicator ningen.UnreadIndication
}

// NewButton creates a new button.
func NewButton(ctx context.Context, open func()) *Button {
	g := Button{
		ctx: ctx,
	}

	g.Icon = avatar.New(ctx, 16, "")
	g.Mentions = NewMentionsIndicator()

	g.IconOverlay = gtk.NewOverlay()
	g.IconOverlay.SetChild(g.Icon.Avatar)
	g.IconOverlay.AddOverlay(g.Mentions)

	g.Button = gtk.NewButton()
	g.Button.SetHasFrame(false)
	g.Button.SetHAlign(gtk.AlignCenter)
	g.Button.SetChild(g.IconOverlay)
	g.Button.ConnectClicked(func() {
		open()
	})

	// iconAnimation := g.Icon.EnableAnimation()
	// iconAnimation.ConnectMotion(g.Button)

	g.Overlay = gtk.NewOverlay()
	g.Overlay.SetChild(g.Button)

	return &g
}

// Context returns the context of the button that was passed in during
// construction.
func (g *Button) Context() context.Context {
	return g.ctx
}

// Activate activates the button.
func (g *Button) Activate() bool {
	return g.Button.Activate()
}

// SetMentions sets the button's mention indicator.
func (g *Button) SetMentions(mentions int) {
	if g.mentions == mentions {
		return
	}

	g.mentions = mentions
}
