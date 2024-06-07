package main

import (
	"context"
	"io"
	"mime"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/chatkit/components/autocomplete"
	"github.com/diamondburned/gotk4/pkg/core/gioutil"
	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/app"
	"github.com/diamondburned/gotkit/app/prefs"
	"github.com/diamondburned/gotkit/gtkutil"
	"github.com/diamondburned/gotkit/gtkutil/textutil"
	"github.com/diamondburned/gotkit/utils/osutil"
	"github.com/nbd-wtf/go-nostr/nip29"
	"github.com/pkg/errors"
)

var persistInput = prefs.NewBool(true, prefs.PropMeta{
	Name:    "Persist Input",
	Section: "Composer",
	Description: "Persist the input message between sessions (to disk). " +
		"If disabled, the input is only persisted for the current session on memory.",
})

// Input is the text field of the composer.
type Input struct {
	*gtk.TextView
	Buffer *gtk.TextBuffer
	ac     *autocomplete.Autocompleter

	ctx  context.Context
	ctrl *ComposerView
	gad  nip29.GroupAddress
}

// inputStateKey is the app state that stores the last input message.
var inputStateKey = app.NewStateKey[string]("input-state")

var inputStateMemory sync.Map // map[string]string

func NewInput(ctx context.Context, ctrl *ComposerView, gad nip29.GroupAddress) *Input {
	i := Input{
		ctx:  ctx,
		ctrl: ctrl,
		gad:  gad,
	}

	i.TextView = gtk.NewTextView()
	i.TextView.AddCSSClass("mx-2")
	i.TextView.AddCSSClass("px-4")
	i.TextView.AddCSSClass("py-2")
	i.TextView.SetWrapMode(gtk.WrapWordChar)
	i.TextView.SetAcceptsTab(true)
	i.TextView.SetHExpand(true)
	i.TextView.SetInputHints(0 |
		gtk.InputHintEmoji |
		gtk.InputHintSpellcheck |
		gtk.InputHintWordCompletion |
		gtk.InputHintUppercaseSentences,
	)
	textutil.SetTabSize(i.TextView)

	i.TextView.ConnectPasteClipboard(i.readClipboard)

	i.ac = autocomplete.New(ctx, i.TextView)
	i.ac.AddSelectedFunc(i.onAutocompleted)
	i.ac.SetCancelOnChange(false)
	i.ac.SetMinLength(2)
	i.ac.SetTimeout(time.Second)

	// state := gtkcord.FromContext(ctx)
	// if ch, err := state.Cabinet.Channel(chID); err == nil {
	// 	i.guildID = ch.GuildID
	// 	i.ac.Use(
	// 		NewMemberCompleter(chID), // @
	// 	)
	// }

	inputState := inputStateKey.Acquire(ctx)

	i.Buffer = i.TextView.Buffer()
	i.Buffer.ConnectChanged(func() {
		i.ac.Autocomplete()

		start, end := i.Buffer.Bounds()

		// Persist input.
		if end.Offset() == 0 {
			if persistInput.Value() {
				inputState.Delete(gad.String())
			} else {
				inputStateMemory.Delete(gad.String())
			}
		} else {
			text := i.Buffer.Text(start, end, false)
			if persistInput.Value() {
				inputState.Set(gad.String(), text)
			} else {
				inputStateMemory.Store(gad.String(), text)
			}
		}
	})

	enterKeyer := gtk.NewEventControllerKey()
	enterKeyer.ConnectKeyPressed(i.onKey)
	i.AddController(enterKeyer)

	inputState.Get(gad.String(), func(text string) {
		i.Buffer.SetText(text)
	})

	return &i
}

func (i *Input) onAutocompleted(row autocomplete.SelectedData) bool {
	i.Buffer.BeginUserAction()
	defer i.Buffer.EndUserAction()

	i.Buffer.Delete(row.Bounds[0], row.Bounds[1])

	switch data := row.Data.(type) {
	case MemberData:
		i.Buffer.Insert(row.Bounds[1], discord.Member(data).Mention())
		return true
	}

	return false
}

var sendOnEnter = prefs.NewBool(true, prefs.PropMeta{
	Name:        "Send Message on Enter",
	Section:     "Composer",
	Description: "Send the message when the user hits the Enter key. Disable this for mobile.",
})

func (i *Input) onKey(val, _ uint, mod gdk.ModifierType) bool {
	switch val {
	case gdk.KEY_Return, gdk.KEY_KP_Enter:
		if i.ac.Select() {
			return true
		}

		if sendOnEnter.Value() && !mod.Has(gdk.ShiftMask) {
			i.ctrl.publish()
			return true
		}
	case gdk.KEY_Tab:
		return i.ac.Select()
	case gdk.KEY_Escape:
		if i.ctrl.replyingTo != "" {
			i.ctrl.StopReplying()
			return true
		}
	case gdk.KEY_Up:
		if i.ac.MoveUp() {
			return true
		}
	case gdk.KEY_Down:
		return i.ac.MoveDown()
	}

	return false
}

func (i *Input) readClipboard() {
	display := gdk.DisplayGetDefault()

	clipboard := display.Clipboard()
	mimeTypes := clipboard.Formats().MIMETypes()

	// Ignore anything text.
	for _, mime := range mimeTypes {
		if mimeIsText(mime) {
			return
		}
	}

	clipboard.ReadAsync(i.ctx, mimeTypes, int(glib.PriorityDefault), func(res gio.AsyncResulter) {
		typ, streamer, err := clipboard.ReadFinish(res)
		if err != nil {
			app.Error(i.ctx, errors.Wrap(err, "failed to read clipboard"))
			return
		}

		gtkutil.Async(i.ctx, func() func() {
			stream := gio.BaseInputStream(streamer)
			reader := gioutil.Reader(i.ctx, stream)
			defer reader.Close()

			f, err := osutil.Consume(reader)
			if err != nil {
				app.Error(i.ctx, errors.Wrap(err, "cannot clone clipboard"))
				return nil
			}

			s, err := f.Stat()
			if err != nil {
				app.Error(i.ctx, errors.Wrap(err, "cannot stat clipboard file"))
				return nil
			}

			// We're too lazy to do reference-counting, so just forbid Open from
			// being called more than once.
			var openedOnce bool

			file := File{
				Name: "clipboard",
				Type: typ,
				Size: s.Size(),
				Open: func() (io.ReadCloser, error) {
					if !openedOnce {
						openedOnce = true
						return f, nil
					}
					return nil, errors.New("Open called more than once on TempFile")
				},
			}

			if exts, _ := mime.ExtensionsByType(typ); len(exts) > 0 {
				file.Name += exts[0]
			}

			return func() {
				i.ctrl.UploadTray.AddFile(file)
			}
		})
	})
}
