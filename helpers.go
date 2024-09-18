package main

import (
	"iter"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type gtkparent interface {
	LastChild() gtk.Widgetter
}

type gtkchild interface {
	PrevSibling() gtk.Widgetter
}

func children[P gtkparent, C gtkchild](list P) iter.Seq[C] {
	return func(yield func(C) bool) {
		row := list.LastChild()
		if row == nil {
			return
		}

		c := row.(C)
		for {
			// this repeats until index is -1, at which the loop will break.
			prev := c.PrevSibling()
			if prev == nil {
				yield(c)
				return
			}

			if !yield(c) {
				return
			}

			c = prev.(C)
		}
	}
}

func trimProtocol(relay string) string {
	relay = strings.TrimPrefix(relay, "wss://")
	relay = strings.TrimPrefix(relay, "ws://")
	relay = strings.TrimPrefix(relay, "wss:/") // Some browsers replace upfront '//' with '/'
	relay = strings.TrimPrefix(relay, "ws:/")  // Some browsers replace upfront '//' with '/'
	return relay
}

func fixNatWrap(label *gtk.Label) {
	if err := gtk.CheckVersion(4, 6, 0); err == "" {
		label.SetObjectProperty("natural-wrap-mode", 1) // NaturalWrapNone
	}
}
