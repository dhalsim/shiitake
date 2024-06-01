package main

import (
	"strings"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func eachChild(list *gtk.ListBox, fn func(*gtk.ListBoxRow) bool) {
	row, _ := list.LastChild().(*gtk.ListBoxRow)
	for row != nil {
		if fn(row) {
			break
		}
		// This repeats until index is -1, at which the loop will break.
		row, _ = row.PrevSibling().(*gtk.ListBoxRow)
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
