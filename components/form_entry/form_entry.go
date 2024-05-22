package form_entry

import (
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
)

// FormEntry is a widget containing a label and an entry.
type FormEntry struct {
	*gtk.Box
	Label *gtk.Label
	Entry *gtk.Entry
}

var formEntryCSS = cssutil.Applier("login-formentry", ``)

// New creates a new FormEntry.
func New(label string) *FormEntry {
	e := FormEntry{}
	e.Label = gtk.NewLabel(label)
	e.Label.SetXAlign(0)

	e.Entry = gtk.NewEntry()
	e.Entry.SetVExpand(true)
	e.Entry.SetHasFrame(true)

	e.Box = gtk.NewBox(gtk.OrientationVertical, 0)
	e.Box.Append(e.Label)
	e.Box.Append(e.Entry)
	formEntryCSS(e)

	return &e
}

// Text gets the value entry.
func (e *FormEntry) Text() string { return e.Entry.Text() }

// FocusNext navigates to the next widget.
func (e *FormEntry) FocusNext() {
	e.Entry.Emit("move-focus", gtk.DirTabForward)
}

// FocusNextOnActivate binds Enter to navigate to the next widget when it's
// pressed.
func (e *FormEntry) FocusNextOnActivate() {
	e.Entry.ConnectActivate(e.FocusNext)
}

// ConnectActivate connects the activate signal hanlder to the Entry.
func (e *FormEntry) ConnectActivate(f func()) {
	e.Entry.ConnectActivate(f)
}
