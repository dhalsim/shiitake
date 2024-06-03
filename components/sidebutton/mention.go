package sidebutton

import (
	"strconv"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// MentionsIndicator is a small indicator that shows the mention count.
type MentionsIndicator struct {
	*gtk.Revealer
	Label *gtk.Label

	count  int
	reveal bool
}

// NewMentionsIndicator creates a new mention indicator.
func NewMentionsIndicator() *MentionsIndicator {
	m := &MentionsIndicator{
		Revealer: gtk.NewRevealer(),
		Label:    gtk.NewLabel(""),
		reveal:   true,
	}

	m.SetChild(m.Label)
	m.SetHAlign(gtk.AlignEnd)
	m.SetVAlign(gtk.AlignEnd)
	m.SetTransitionType(gtk.RevealerTransitionTypeCrossfade)
	m.SetTransitionDuration(100)

	m.update()
	return m
}

// SetCount sets the mention count.
func (m *MentionsIndicator) SetCount(count int) {
	if count == m.count {
		return
	}

	m.count = count
	m.update()
}

// Count returns the mention count.
func (m *MentionsIndicator) Count() int {
	return m.count
}

// SetRevealChild sets whether the indicator should be revealed.
// This lets the user hide the indicator even if there are mentions.
func (m *MentionsIndicator) SetRevealChild(reveal bool) {
	m.reveal = reveal
	m.update()
}

func (m *MentionsIndicator) update() {
	if m.count == 0 {
		m.RemoveCSSClass("sidebar-mention-active")
		m.Revealer.SetRevealChild(false)
		return
	}

	m.Label.SetText(strconv.Itoa(m.count))
	m.Revealer.SetRevealChild(m.reveal && m.count > 0)
}
