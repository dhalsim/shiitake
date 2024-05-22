package groups

import (
	"context"
	"fmt"

	"fiatjaf.com/shiitake/global"
	"fiatjaf.com/shiitake/signaling"
	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/nbd-wtf/go-nostr/nip29"
)

type modelManager struct {
	*gtk.TreeListModel
	relayID string
}

func newModelManager(relayID string) *modelManager {
	m := &modelManager{
		relayID: relayID,
	}
	m.TreeListModel = gtk.NewTreeListModel(
		m.Model(nip29.GroupAddress{}), true, true,
		func(item *glib.Object) *gio.ListModel {
			fmt.Println("blub")
			gad := gadFromItem(item)
			fmt.Println("  gad", gad)

			model := m.Model(gad)
			fmt.Println("    model", model)
			if model == nil {
				return nil
			}

			return &model.ListModel
		})
	return m
}

// Model returns the list model containing all groups within the given group
// ID. If gad is 0, then the relay's root groups will be returned. This
// function may return nil, indicating that the group will never have any
// children.
func (m *modelManager) Model(gad nip29.GroupAddress) *gtk.StringList {
	model := gtk.NewStringList(nil)

	list := newGroupList(model)

	var unbind signaling.DisconnectStack
	list.ConnectDestroy(func() { unbind.Disconnect() })

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		me := global.GetMe(ctx)
		for {
			select {
			case group := <-me.JoinedGroup:
				list.Append(group.Address)
			case gad := <-me.LeftGroup:
				list.Remove(gad)
			}
		}
	}()

	unbind.Push(cancel)
	return model
}

// groupList wraps a StringList to maintain a set of group IDs.
// Because this is a set, each group ID can only appear once.
type groupList struct {
	list *gtk.StringList
	ids  []nip29.GroupAddress
}

func newGroupList(list *gtk.StringList) *groupList {
	return &groupList{
		list: list,
		ids:  make([]nip29.GroupAddress, 0, 4),
	}
}

// Append appends a group to the list. If the group already exists, then
// this function does nothing.
func (l *groupList) Append(gad nip29.GroupAddress) {
	l.ids = append(l.ids, gad)
}

// Remove removes the group ID from the list. If the group ID is not in the
// list, then this function does nothing.
func (l *groupList) Remove(gad nip29.GroupAddress) {
	i := l.Index(gad)
	if i != -1 {
		l.ids = append(l.ids[:i], l.ids[i+1:]...)

		list := l.list
		if list != nil {
			list.Remove(uint(i))
		}
	}
}

// Contains returns whether the group ID is in the list.
func (l *groupList) Contains(gad nip29.GroupAddress) bool {
	return l.Index(gad) != -1
}

// Index returns the index of the group ID in the list. If the group ID is
// not in the list, then this function returns -1.
func (l *groupList) Index(gad nip29.GroupAddress) int {
	for i, c := range l.ids {
		if c.Equals(gad) {
			return i
		}
	}
	return -1
}

// Clear clears the list.
func (l *groupList) Clear() {
	l.ids = l.ids[:0]

	list := l.list
	if list != nil {
		list.Splice(0, list.NItems(), nil)
	}
}

func (l *groupList) ConnectDestroy(f func()) {
	list := l.list
	if list == nil {
		return
	}
	// I think this is the only way to know if a ListModel is no longer
	// being used? At least from reading the source code, which just calls
	// g_clear_pointer.
	glib.WeakRefObject(list, f)
}
