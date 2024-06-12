package main

import (
	"strings"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func eachChild(list *gtk.ListBox, fn func(*gtk.ListBoxRow) bool) {
	row, _ := list.LastChild().(*gtk.ListBoxRow)
	for row != nil {
		// this repeats until index is -1, at which the loop will break.
		prev, _ := row.PrevSibling().(*gtk.ListBoxRow)

		if fn(row) {
			break
		}

		row = prev
	}
}

func getChild(list *gtk.ListBox, fn func(*gtk.ListBoxRow) bool) *gtk.ListBoxRow {
	var row *gtk.ListBoxRow
	eachChild(list, func(lbr *gtk.ListBoxRow) bool {
		if fn(lbr) {
			// found it
			row = lbr
			return true
		}
		return false
	})
	return row
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
