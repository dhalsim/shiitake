package groups

import (
	"context"
	"fmt"

	"fiatjaf.com/shiitake/components/hoverpopover"
	"fiatjaf.com/shiitake/signaling"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/chatkit/components/author"
	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gotkit/app"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
	"github.com/diamondburned/gotkit/gtkutil/imgutil"
	"github.com/nbd-wtf/go-nostr/nip29"
)

var revealStateKey = app.NewStateKey[bool]("collapsed-groups-state")

type groupItemState struct {
	reveal *app.TypedState[bool]
}

func newGroupItemFactory(ctx context.Context, model *groupsModelManager) *gtk.ListItemFactory {
	factory := gtk.NewSignalListItemFactory()
	state := groupItemState{
		reveal: revealStateKey.Acquire(ctx),
	}

	unbindFns := make(map[uintptr]func())

	factory.ConnectBind(func(item *gtk.ListItem) {
		row := model.Row(item.Position())
		unbind := bindGroupItem(state, item, row)
		unbindFns[item.Native()] = unbind
	})

	factory.ConnectUnbind(func(item *gtk.ListItem) {
		unbind := unbindFns[item.Native()]
		unbind()
		delete(unbindFns, item.Native())
		item.SetChild(nil)
	})

	return &factory.ListItemFactory
}

func gadFromListItem(item *gtk.ListItem) nip29.GroupAddress {
	return gadFromItem(item.Item())
}

func gadFromItem(item *glib.Object) nip29.GroupAddress {
	str := item.Cast().(*gtk.StringObject)

	gad, err := nip29.ParseGroupAddress(str.String())
	if err != nil {
		panic(fmt.Sprintf("gadFromListItem: failed to parse gad: %v", err))
	}

	return gad
}

var _ = cssutil.WriteCSS(`
	.groups-viewtree row:hover,
	.groups-viewtree row:selected {
		background: none;
	}
	.groups-viewtree row:hover .group-item-outer {
		background: alpha(@theme_fg_color, 0.075);
	}
	.groups-viewtree row:selected .group-item-outer {
		background: alpha(@theme_fg_color, 0.125);
	}
	.groups-viewtree row:selected:hover .group-item-outer {
		background: alpha(@theme_fg_color, 0.175);
	}
	.group-item {
		padding: 0.35em 0;
	}
	.group-item :first-child {
		min-width: 2.5em;
		margin: 0;
	}
	.group-item expander + * {
		/* Weird workaround because GTK is adding extra padding here for some
		 * reason. */
		margin-left: -0.35em;
	}
	.group-item-muted {
		opacity: 0.35;
	}
	.group-unread-indicator {
		font-size: 0.75em;
		font-weight: 700;
	}
	.group-item-unread .group-unread-indicator,
	.group-item-mentioned .group-unread-indicator {
		font-size: 0.7em;
		font-weight: 900;
		font-family: monospace;

		min-width: 1em;
		min-height: 1em;
		line-height: 1em;

		padding: 0;
		margin: 0 1em;

		outline: 1.5px solid @theme_fg_color;
		border-radius: 99px;
	}
	.group-item-mentioned .group-unread-indicator {
		font-size: 0.8em;
		outline-color: @mentioned;
		background: @mentioned;
		color: @theme_bg_color;
	}
`)

type groupItem struct {
	item   *gtk.ListItem
	row    *gtk.TreeListRow
	reveal *app.TypedState[bool]

	child struct {
		*gtk.Box
		content   gtk.Widgetter
		indicator *gtk.Label
	}

	gad nip29.GroupAddress
}

func bindGroupItem(state groupItemState, item *gtk.ListItem, row *gtk.TreeListRow) func() {
	i := &groupItem{
		item:   item,
		row:    row,
		reveal: state.reveal,
		gad:    gadFromListItem(item),
	}

	i.child.indicator = gtk.NewLabel("")
	i.child.indicator.AddCSSClass("group-unread-indicator")
	i.child.indicator.SetHExpand(true)
	i.child.indicator.SetHAlign(gtk.AlignEnd)
	i.child.indicator.SetVAlign(gtk.AlignCenter)

	i.child.Box = gtk.NewBox(gtk.OrientationHorizontal, 0)
	i.child.Box.Append(i.child.indicator)

	hoverpopover.NewMarkupHoverPopover(i.child.Box, func(w *hoverpopover.MarkupHoverPopoverWidget) bool {
		// summary := i.state.SummaryState.LastSummary(i.gad)
		// if summary == nil {
		// 	return false
		// }

		// window := app.GTKWindowFromContext(i.state.Context())
		// if window.Width() > 600 {
		// 	w.SetPosition(gtk.PosRight)
		// } else {
		// 	w.SetPosition(gtk.PosBottom)
		// }

		// w.Label.SetEllipsize(pango.EllipsizeEnd)
		// w.Label.SetSingleLineMode(true)
		// w.Label.SetMaxWidthChars(50)
		// w.Label.SetMarkup(fmt.Sprintf(
		// 	"<b>%s</b>%s",
		// 	locale.Get("Chatting about: "),
		// 	summary.Topic,
		// ))

		return true
	})

	i.item.SetChild(i.child.Box)

	var unbind signaling.DisconnectStack
	// unbind.Push(
	// 	i.state.AddHandler(func(ev *read.UpdateEvent) {
	// 		if ev.GroupID == i.gad {
	// 			i.Invalidate()
	// 		}
	// 	}),
	// 	i.state.AddHandler(func(ev *gateway.GroupUpdateEvent) {
	// 		if ev.ID == i.gad {
	// 			i.Invalidate()
	// 		}
	// 	}),
	// )

	// ch, _ := i.state.Offline().Group(i.gad)
	// if ch != nil {
	// 	switch ch.Type {
	// 	case discord.RelayPublicThread, discord.RelayPrivateThread, discord.RelayAnnouncementThread:
	// 		unbind.Push(i.state.AddHandler(func(ev *gateway.ThreadUpdateEvent) {
	// 			if ev.ID == i.gad {
	// 				i.Invalidate()
	// 			}
	// 		}))
	// 	}

	// 	relayID := ch.RelayID
	// 	switch ch.Type {
	// 	case discord.RelayVoice, discord.RelayStageVoice:
	// 		unbind.Push(i.state.AddHandler(func(ev *gateway.VoiceStateUpdateEvent) {
	// 			// The group ID becomes null when the user leaves the group,
	// 			// so we'll just update when any relay state changes.
	// 			if ev.RelayID == relayID {
	// 				i.Invalidate()
	// 			}
	// 		}))
	// 	}
	// }

	i.Invalidate()
	return unbind.Disconnect
}

type readStatus = int

const (
	unread    readStatus = iota
	read      readStatus = iota
	mentioned readStatus = iota
)

var readCSSClasses = map[readStatus]string{
	unread:    "group-item-unread",
	mentioned: "group-item-mentioned",
}

const groupMutedClass = "group-item-muted"

// Invalidate updates the group item's contents.
func (i *groupItem) Invalidate() {
	if i.child.content != nil {
		i.child.Box.Remove(i.child.content)
	}

	i.item.SetSelectable(true)

	// ch, _ := i.state.Offline().Group(i.gad)
	// if ch == nil {
	// 	i.child.content = newUnknownGroupItem(i.gad.String())
	// 	i.item.SetSelectable(false)
	// } else {
	// 	switch ch.Type {
	// 	case
	// 		discord.RelayText, discord.RelayAnnouncement,
	// 		discord.RelayPublicThread, discord.RelayPrivateThread, discord.RelayAnnouncementThread:

	// 		i.child.content = newGroupItemText(ch)

	// 	case discord.RelayCategory, discord.RelayForum:
	// 		switch ch.Type {
	// 		case discord.RelayCategory:
	// 			i.child.content = newGroupItemCategory(ch, i.row, i.reveal)
	// 			i.item.SetSelectable(false)
	// 		case discord.RelayForum:
	// 			i.child.content = newGroupItemForum(ch, i.row)
	// 		}

	// 	case discord.RelayVoice, discord.RelayStageVoice:
	// 		i.child.content = newGroupItemVoice(i.state, ch)

	// 	default:
	// 		panic("unreachable")
	// 	}
	// }

	// i.child.Box.SetCSSClasses(nil)
	// i.child.Box.Prepend(i.child.content)

	// // Steal CSS classes from the child.
	// for _, class := range gtk.BaseWidget(i.child.content).CSSClasses() {
	// 	i.child.Box.AddCSSClass(class + "-outer")
	// }

	// unreadOpts := ningen.UnreadOpts{
	// 	// We can do this within the group list itself because it's easy to
	// 	// expand categories and see the unread groups within them.
	// 	IncludeMutedCategories: true,
	// }

	// unread := i.state.GroupIsUnread(i.gad, unreadOpts)
	// if unread != ningen.GroupRead {
	// 	i.child.Box.AddCSSClass(readCSSClasses[unread])
	// }

	// i.updateIndicator(unread)

	// if i.state.GroupIsMuted(i.gad, unreadOpts) {
	// 	i.child.Box.AddCSSClass(groupMutedClass)
	// } else {
	// 	i.child.Box.RemoveCSSClass(groupMutedClass)
	// }
}

func (i *groupItem) updateIndicator(s readStatus) {
	if unread == mentioned {
		i.child.indicator.SetText("!")
	} else {
		i.child.indicator.SetText("")
	}
}

var _ = cssutil.WriteCSS(`
	.group-item-unknown {
		opacity: 0.35;
		font-style: italic;
	}
`)

func newUnknownGroupItem(name string) gtk.Widgetter {
	icon := gtk.NewImageFromIconName("group-symbolic")

	label := gtk.NewLabel(name)
	label.SetEllipsize(pango.EllipsizeEnd)
	label.SetXAlign(0)

	box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	box.AddCSSClass("group-item")
	box.AddCSSClass("group-item-unknown")
	box.Append(icon)
	box.Append(label)

	return box
}

var _ = cssutil.WriteCSS(`
	.group-item-thread {
		padding: 0.25em 0;
		opacity: 0.5;
	}
	.group-item-unread .group-item-thread,
	.group-item-mention .group-item-thread {
		opacity: 1;
	}
	.group-item-nsfw-indicator {
		font-size: 0.75em;
		font-weight: bold;
		margin-right: 0.75em;
	}
`)

func newGroupItemText(ch *nip29.Group) gtk.Widgetter {
	icon := gtk.NewImageFromIconName("")

	icon.SetFromIconName("group-symbolic")
	// switch ch.Type {
	// case discord.RelayText:
	// 	icon.SetFromIconName("group-symbolic")
	// case discord.RelayAnnouncement:
	// 	icon.SetFromIconName("group-broadcast-symbolic")
	// case discord.RelayPublicThread, discord.RelayPrivateThread, discord.RelayAnnouncementThread:
	// 	icon.SetFromIconName("thread-branch-symbolic")
	// }

	iconFrame := gtk.NewOverlay()
	iconFrame.SetChild(icon)

	// if ch.NSFW {
	// 	nsfwIndicator := gtk.NewLabel("!")
	// 	nsfwIndicator.AddCSSClass("group-item-nsfw-indicator")
	// 	nsfwIndicator.SetHAlign(gtk.AlignEnd)
	// 	nsfwIndicator.SetVAlign(gtk.AlignEnd)
	// 	iconFrame.AddOverlay(nsfwIndicator)
	// }

	label := gtk.NewLabel(ch.Name)
	label.SetEllipsize(pango.EllipsizeEnd)
	label.SetXAlign(0)
	bindLabelTooltip(label, false)

	box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	box.AddCSSClass("group-item")
	box.Append(iconFrame)
	box.Append(label)

	box.AddCSSClass("group-item-text")

	// switch ch.Type {
	// case discord.RelayText:
	// 	box.AddCSSClass("group-item-text")
	// case discord.RelayAnnouncement:
	// 	box.AddCSSClass("group-item-announcement")
	// case discord.RelayPublicThread, discord.RelayPrivateThread, discord.RelayAnnouncementThread:
	// 	box.AddCSSClass("group-item-thread")
	// }

	return box
}

var _ = cssutil.WriteCSS(`
	.group-item-forum {
		padding: 0.35em 0;
	}
	.group-item-forum label {
		padding: 0;
	}
`)

func newGroupItemForum(ch *nip29.Group, row *gtk.TreeListRow) gtk.Widgetter {
	label := gtk.NewLabel(ch.Name)
	label.SetEllipsize(pango.EllipsizeEnd)
	label.SetXAlign(0)
	bindLabelTooltip(label, false)

	expander := gtk.NewTreeExpander()
	expander.AddCSSClass("group-item")
	expander.AddCSSClass("group-item-forum")
	expander.SetHExpand(true)
	expander.SetListRow(row)
	expander.SetChild(label)

	// GTK 4.10 or later only.
	expander.SetObjectProperty("indent-for-depth", false)

	return expander
}

var _ = cssutil.WriteCSS(`
	.groups-viewtree row:not(:first-child) .group-item-category-outer {
		margin-top: 0.75em;
	}
	.groups-viewtree row:hover .group-item-category-outer {
		background: none;
	}
	.group-item-category {
		padding: 0.4em 0;
	}
	.group-item-category label {
		margin-bottom: -0.2em;
		padding: 0;
		font-size: 0.85em;
		font-weight: 700;
		text-transform: uppercase;
	}
`)

func newGroupItemCategory(ch *nip29.Group, row *gtk.TreeListRow, reveal *app.TypedState[bool]) gtk.Widgetter {
	label := gtk.NewLabel(ch.Name)
	label.SetEllipsize(pango.EllipsizeEnd)
	label.SetXAlign(0)
	bindLabelTooltip(label, false)

	expander := gtk.NewTreeExpander()
	expander.AddCSSClass("group-item")
	expander.AddCSSClass("group-item-category")
	expander.SetHExpand(true)
	expander.SetListRow(row)
	expander.SetChild(label)

	ref := glib.NewWeakRef[*gtk.TreeListRow](row)
	gad := ch.Address

	// Add this notifier after a small delay so GTK can initialize the row.
	// Otherwise, it will falsely emit the signal.
	glib.TimeoutSecondsAdd(1, func() {
		row := ref.Get()
		if row == nil {
			return
		}

		row.NotifyProperty("expanded", func() {
			row := ref.Get()
			if row == nil {
				return
			}

			// Only retain collapsed states. Expanded states are assumed to be
			// the default.
			if !row.Expanded() {
				reveal.Set(gad.String(), true)
			} else {
				reveal.Delete(gad.String())
			}
		})
	})

	reveal.Get(gad.String(), func(collapsed bool) {
		if collapsed {
			// GTK will actually explode if we set the expanded property without
			// waiting for it to load for some reason?
			glib.IdleAdd(func() { row.SetExpanded(false) })
		}
	})

	return expander
}

var _ = cssutil.WriteCSS(`
	.group-item-voice .mauthor-chip {
		margin: 0.15em 0;
		margin-left: 2.5em;
		margin-right: 1em;
	}
	.group-item-voice .mauthor-chip:nth-child(2) {
		margin-top: 0;
	}
	.group-item-voice .mauthor-chip:last-child {
		margin-bottom: 0.3em;
	}
	.group-item-voice-counter {
		margin-left: 0.5em;
		margin-right: 0.5em;
		font-size: 0.8em;
		opacity: 0.75;
	}
`)

func newGroupItemVoice(ch *nip29.Group) gtk.Widgetter {
	icon := gtk.NewImageFromIconName("group-voice-symbolic")

	label := gtk.NewLabel(ch.Name)
	label.SetEllipsize(pango.EllipsizeEnd)
	label.SetXAlign(0)
	label.SetTooltipText(ch.Name)

	top := gtk.NewBox(gtk.OrientationHorizontal, 0)
	top.AddCSSClass("group-item")
	top.Append(icon)
	top.Append(label)

	// var voiceParticipants int
	// voiceStates, _ := state.VoiceStates(ch.RelayID)
	// for _, voiceState := range voiceStates {
	// 	if voiceState.GroupID == ch.ID {
	// 		voiceParticipants++
	// 	}
	// }

	// if voiceParticipants > 0 {
	// 	counter := gtk.NewLabel(fmt.Sprintf("%d", voiceParticipants))
	// 	counter.AddCSSClass("group-item-voice-counter")
	// 	counter.SetVExpand(true)
	// 	counter.SetXAlign(0)
	// 	counter.SetYAlign(1)
	// 	top.Append(counter)
	// }

	return top

	// TODO: fix read indicator alignment. This probably should be in a separate
	// ListModel instead.

	// box := gtk.NewBox(gtk.OrientationVertical, 0)
	// box.AddCSSClass("group-item-voice")
	// box.Append(top)

	// voiceStates, _ := state.VoiceStates(ch.RelayID)
	// for _, voiceState := range voiceStates {
	// 	if voiceState.GroupID == ch.ID {
	// 		box.Append(newVoiceParticipant(state, voiceState))
	// 	}
	// }

	// return box
}

func newVoiceParticipant(voiceState discord.VoiceState) gtk.Widgetter {
	chip := author.NewChip(context.Background(), imgutil.HTTPProvider)
	chip.Unpad()

	member := voiceState.Member
	// if member == nil {
	// 	member, _ = state.Member(voiceState.RelayID, voiceState.UserID)
	// }

	if member != nil {
		chip.SetName(member.User.DisplayOrUsername())
		// chip.SetAvatar(gtkcord.InjectAvatarSize(member.AvatarURL(voiceState.RelayID)))
		// if color, ok := state.MemberColor(voiceState.RelayID, voiceState.UserID); ok {
		// 	chip.SetColor(color.String())
		// }
	} else {
		chip.SetName(voiceState.UserID.String())
	}

	return chip
}

func bindLabelTooltip(label *gtk.Label, markup bool) {
	ref := glib.NewWeakRef(label)
	label.NotifyProperty("label", func() {
		label := ref.Get()
		inner := label.Label()
		if markup {
			label.SetTooltipMarkup(inner)
		} else {
			label.SetTooltipText(inner)
		}
	})
}
