package groups

import (
	"context"
	"fmt"
	"slices"

	"fiatjaf.com/shiitake/global"
	"fiatjaf.com/shiitake/signaling"
	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/nbd-wtf/go-nostr/nip29"
)

type groupsModelManager struct {
	*gtk.TreeListModel
	relayURL string
}

func newGroupsModelManager(relayURL string) *groupsModelManager {
	m := &groupsModelManager{
		relayURL: relayURL,
	}

	gmodel := groupsModel(nip29.GroupAddress{})

	m.TreeListModel = gtk.NewTreeListModel(
		gmodel.model,
		true,
		true,
		func(item *glib.Object) (listModel *gio.ListModel) { return nil },
	)

	return m
}

// groupsModel returns the list model containing all groups within the given relay
// If gad is 0, then the relay's root groups will be returned. This
// function may return nil, indicating that the group will never have any
// children.
func groupsModel(gad nip29.GroupAddress) *groupList {
	model := gtk.NewStringList(nil)

	gl := &groupList{
		model: model,
		ids:   make([]nip29.GroupAddress, 0, 4),
	}

	var unbind signaling.DisconnectStack
	gl.ConnectDestroy(func() { unbind.Disconnect() })

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		me := global.GetMe(ctx)
		for {
			select {
			case group := <-me.JoinedGroup:
				fmt.Println("jjj", group)
				gl.Append(group.Address)
			case gad := <-me.LeftGroup:
				gl.Remove(gad)
			}
		}
	}()

	unbind.Push(cancel)
	return gl
}

// groupList wraps a StringList to maintain a set of group IDs.
// Because this is a set, each group ID can only appear once.
type groupList struct {
	model *gtk.StringList
	ids   []nip29.GroupAddress
}

// Append appends a group to the list. If the group already exists, then
// this function does nothing.
func (l *groupList) Append(gad nip29.GroupAddress) {
	l.model.Append(gad.String())
	l.ids = append(l.ids, gad)
}

// Remove removes the group ID from the list. If the group ID is not in the
// list, then this function does nothing.
func (l *groupList) Remove(gad nip29.GroupAddress) {
	i := l.Index(gad)
	if i == -1 {
		return
	}

	l.ids = slices.Delete(l.ids, i, i+1)
	l.ids = append(l.ids[:i], l.ids[i+1:]...)

	l.model.Remove(uint(i))
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

	list := l.model
	if list != nil {
		list.Splice(0, list.NItems(), nil)
	}
}

func (l *groupList) ConnectDestroy(f func()) {
	list := l.model
	if list == nil {
		return
	}
	// I think this is the only way to know if a ListModel is no longer
	// being used? At least from reading the source code, which just calls
	// g_clear_pointer.
	glib.WeakRefObject(list, f)
}
