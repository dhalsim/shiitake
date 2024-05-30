package main

import "github.com/diamondburned/gotk4/pkg/gtk/v4"

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
