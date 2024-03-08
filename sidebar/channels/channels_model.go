package channels

import (
	"fiatjaf.com/shiitake/signaling"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type modelManager struct {
	*gtk.TreeListModel
	guildID string
}

func newModelManager(guildID string) *modelManager {
	m := &modelManager{
		guildID: guildID,
	}
	m.TreeListModel = gtk.NewTreeListModel(
		m.Model(""), true, true,
		func(item *glib.Object) *gio.ListModel {
			chID := channelIDFromItem(item)

			model := m.Model(chID)
			if model == nil {
				return nil
			}

			return &model.ListModel
		})
	return m
}

// Model returns the list model containing all channels within the given channel
// ID. If chID is 0, then the guild's root channels will be returned. This
// function may return nil, indicating that the channel will never have any
// children.
func (m *modelManager) Model(chID string) *gtk.StringList {
	model := gtk.NewStringList(nil)

	list := newChannelList(model)

	var unbind signaling.DisconnectStack
	list.ConnectDestroy(func() { unbind.Disconnect() })

	// unbind.Push(
	// 	m.state.AddHandler(func(ev *gateway.ChannelCreateEvent) {
	// 		if ev.GuildID != m.guildID {
	// 			return
	// 		}
	// 		if ev.Channel.ParentID == chID {
	// 			list.Append(ev.Channel)
	// 		}
	// 	}),
	// 	m.state.AddHandler(func(ev *gateway.ChannelUpdateEvent) {
	// 		if ev.GuildID != m.guildID {
	// 			return
	// 		}
	// 		// Handle channel position moves.
	// 		if ev.Channel.ParentID == chID {
	// 			list.Append(ev.Channel)
	// 		} else {
	// 			list.Remove(ev.Channel.ID)
	// 		}
	// 	}),
	// 	m.state.AddHandler(func(ev *gateway.ThreadCreateEvent) {
	// 		if ev.GuildID != m.guildID {
	// 			return
	// 		}
	// 		if ev.Channel.ParentID == chID {
	// 			list.Append(ev.Channel)
	// 		}
	// 	}),
	// 	m.state.AddHandler(func(ev *gateway.ThreadDeleteEvent) {
	// 		if ev.GuildID != m.guildID {
	// 			return
	// 		}
	// 		if ev.ParentID == chID {
	// 			list.Remove(ev.ID)
	// 		}
	// 	}),
	// 	m.state.AddHandler(func(ev *gateway.ThreadListSyncEvent) {
	// 		if ev.GuildID != m.guildID {
	// 			return
	// 		}

	// 		if ev.ChannelIDs == nil {
	// 			// The entire guild was synced, so invalidate everything.
	// 			m.invalidateAll(chID, list)
	// 			return
	// 		}

	// 		for _, parentID := range ev.ChannelIDs {
	// 			if parentID == chID {
	// 				// This sync event is also for us.
	// 				m.invalidateAll(chID, list)
	// 				break
	// 			}
	// 		}
	// 	}),
	// )

	m.invalidateAll(chID, list)
	return model
}

func (m *modelManager) invalidateAll(parentID string, list *channelList) {
	channels := fetchSortedChannels(m.guildID, parentID)
	list.ClearAndAppend(channels)
}

// channelList wraps a StringList to maintain a set of channel IDs.
// Because this is a set, each channel ID can only appear once.
type channelList struct {
	list *gtk.StringList
	ids  []string
}

func newChannelList(list *gtk.StringList) *channelList {
	return &channelList{
		list: list,
		ids:  make([]string, 0, 4),
	}
}

// CalculatePosition converts the position of a channel given by Discord to the
// position relative to the list. If the channel is not found, then this
// function returns the end of the list.
func (l *channelList) CalculatePosition(target discord.Channel) uint {
	// for i, id := range l.ids {
	// 	ch, _ := l.state.Offline().Channel(id)
	// 	if ch == nil {
	// 		continue
	// 	}

	// 	if ch.Position > target.Position {
	// 		return uint(i)
	// 	}
	// }

	return uint(len(l.ids))
}

// Append appends a channel to the list. If the channel already exists, then
// this function does nothing.
func (l *channelList) Append(ch discord.Channel) {
	pos := l.CalculatePosition(ch)
	l.insertAt(ch, pos)
}

func (l *channelList) insertAt(ch discord.Channel, pos uint) {
	// i := l.Index(ch.ID)
	// if i != -1 {
	// 	return
	// }

	// list := l.list
	// if list == nil {
	// 	return
	// }

	// list.Splice(pos, 0, []string{ch.ID.String()})
	// l.ids = append(l.ids[:pos], append([]string{ch.ID}, l.ids[pos:]...)...)
}

// Remove removes the channel ID from the list. If the channel ID is not in the
// list, then this function does nothing.
func (l *channelList) Remove(chID string) {
	i := l.Index(chID)
	if i != -1 {
		l.ids = append(l.ids[:i], l.ids[i+1:]...)

		list := l.list
		if list != nil {
			list.Remove(uint(i))
		}
	}
}

// Contains returns whether the channel ID is in the list.
func (l *channelList) Contains(chID string) bool {
	return l.Index(chID) != -1
}

// Index returns the index of the channel ID in the list. If the channel ID is
// not in the list, then this function returns -1.
func (l *channelList) Index(chID string) int {
	for i, id := range l.ids {
		if id == chID {
			return i
		}
	}
	return -1
}

// Clear clears the list.
func (l *channelList) Clear() {
	l.ids = l.ids[:0]

	list := l.list
	if list != nil {
		list.Splice(0, list.NItems(), nil)
	}
}

// ClearAndAppend clears the list and appends the given channels.
func (l *channelList) ClearAndAppend(chs []discord.Channel) {
	list := l.list
	if list == nil {
		return
	}

	ids := make([]string, len(chs))
	l.ids = make([]string, len(chs))

	for i, ch := range chs {
		ids[i] = ch.ID.String()
		l.ids = append(l.ids, ch.ID.String())
	}

	list.Splice(0, list.NItems(), ids)
}

func (l *channelList) ConnectDestroy(f func()) {
	list := l.list
	if list == nil {
		return
	}
	// I think this is the only way to know if a ListModel is no longer
	// being used? At least from reading the source code, which just calls
	// g_clear_pointer.
	glib.WeakRefObject(list, f)
}

func fetchSortedChannels(guildID string, parentID string) []discord.Channel {
	// channels, err := state.Offline().Channels(guildID, gtkcord.AllowedChannelTypes)
	// if err != nil {
	// 	log.Printf("CalculatePosition: failed to get channels: %v", err)
	// 	return nil
	// }

	// // Filter out all channels that are not in the same parent channel.
	// filtered := channels[:0]
	// for i, ch := range channels {
	// 	if ch.ParentID == parentID || (parentID == 0 && !ch.ParentID.IsValid()) {
	// 		filtered = append(filtered, channels[i])
	// 	}
	// }

	// // Sort so that the channels are in increasing order.
	// sort.Slice(filtered, func(i, j int) bool {
	// 	a := filtered[i]
	// 	b := filtered[j]
	// 	if a.Position == b.Position {
	// 		return a.ID < b.ID
	// 	}
	// 	return a.Position < b.Position
	// })

	// return filtered
	return nil
}
