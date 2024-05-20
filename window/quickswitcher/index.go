package quickswitcher

import (
	"context"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
	"github.com/nbd-wtf/go-nostr/nip29"
	"github.com/sahilm/fuzzy"
)

type indexItem interface {
	Row(context.Context) *gtk.ListBoxRow
	String() string
}

type indexItems []indexItem

func (its indexItems) String(i int) string { return its[i].String() }
func (its indexItems) Len() int            { return len(its) }

type groupItem struct {
	group  nip29.Group
	name   string
	search string
}

var voiceTypes = map[discord.ChannelType]bool{
	discord.GuildVoice:      true,
	discord.GuildStageVoice: true,
}

func newGroupItem(group nip29.Group) groupItem {
	item := groupItem{
		group: group,
	}

	if group.Name != "" {
		item.name = group.Name
		// } else if len(ch.DMRecipients) == 1 {
		// 	item.name = ch.DMRecipients[0].Tag()
		// } else {
		// 	item.name = gtkcord.RecipientNames(ch)
	}

	// if threadTypes[ch.Type] {
	// 	parent, _ := state.Cabinet.Channel(ch.ParentID)
	// 	if parent != nil {
	// 		item.name = parent.Name + " › #" + item.name
	// 	}
	// }

	return item
}

func (it groupItem) String() string { return it.search }

const (
	chHash       = `<span face="monospace"><b><span size="x-large" rise="-800">#</span><span size="x-small" rise="-2000">  </span></b></span>`
	chNSFWHash   = `<span face="monospace"><b><span size="x-large" rise="-800">#</span><span size="x-small" rise="-2000">! </span></b></span>`
	chVoiceHash  = `<span face="monospace"><b><span size="x-large" rise="-800">#</span><span size="xx-small" rise="-2000">🔊</span></b></span>`
	chThreadHash = `<span face="monospace"><b><span size="x-large" rise="-800">#</span><span size="x-small" rise="-2000"># </span></b></span>`
)

var channelCSS = cssutil.Applier("quickswitcher-channel", `
	.quickswitcher-channel-icon {
		margin: 2px 12px;
		margin-right: 1px;
		min-width:  {$inline_emoji_size};
		min-height: {$inline_emoji_size};
	}
	.quickswitcher-channel-hash {
		padding-left: 1px; /* account for the NSFW mark */
		margin-right: 7px;
	}
	.quickswitcher-channel-image {
		margin-left: 8px;
		margin-right: 12px;
	}
	.quickswitcher-channel-guildname {
		font-size: 0.9em;
		color: alpha(@theme_fg_color, 0.75);
		margin: 4px;
		margin-left: 18px;
		margin-bottom: calc(4px - 0.1em);
	}
`)

func (it groupItem) Row(ctx context.Context) *gtk.ListBoxRow {
	tooltip := it.name
	tooltip += " (" + it.group.Name + ")"

	box := gtk.NewBox(gtk.OrientationHorizontal, 0)

	row := gtk.NewListBoxRow()
	row.SetTooltipText(tooltip)
	row.SetChild(box)
	channelCSS(row)

	icon := gtk.NewLabel("")
	icon.AddCSSClass("quickswitcher-group-icon")
	icon.AddCSSClass("quickswitcher-group-hash")
	icon.SetHAlign(gtk.AlignCenter)
	icon.SetMarkup(chHash)
	box.Append(icon)

	name := gtk.NewLabel(it.name)
	name.AddCSSClass("quickswitcher-channel-name")
	name.SetHExpand(true)
	name.SetXAlign(0)
	name.SetEllipsize(pango.EllipsizeEnd)

	box.Append(name)

	guildName := gtk.NewLabel(it.group.Address.Relay)
	guildName.AddCSSClass("quickswitcher-channel-guildname")
	guildName.SetEllipsize(pango.EllipsizeEnd)

	box.Append(guildName)

	return row
}

type relayItem struct {
	url string
}

func newGuildItem(url string) relayItem {
	return relayItem{
		url: url,
	}
}

func (it relayItem) String() string { return it.url }

var guildCSS = cssutil.Applier("quickswitcher-guild", `
	.quickswitcher-relay-icon {
		margin: 2px 8px;
		min-width:  {$inline_emoji_size};
		min-height: {$inline_emoji_size};
	}
`)

func (it relayItem) Row(ctx context.Context) *gtk.ListBoxRow {
	row := gtk.NewListBoxRow()
	guildCSS(row)

	// icon := onlineimage.NewAvatar(ctx, imgutil.HTTPProvider, gtkcord.InlineEmojiSize)
	// icon.AddCSSClass("quickswitcher-relay-icon")
	// icon.SetInitials(it.Name)
	// icon.SetFromURL(it.IconURL())
	// icon.SetHAlign(gtk.AlignCenter)

	// anim := icon.EnableAnimation()
	// anim.ConnectMotion(row)

	// name := gtk.NewLabel(it.Name)
	// name.AddCSSClass("quickswitcher-relay-name")
	// name.SetHExpand(true)
	// name.SetXAlign(0)

	// box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	// box.Append(icon)
	// box.Append(name)

	// row.SetChild(box)
	return row
}

type index struct {
	items  indexItems
	buffer indexItems
}

const searchLimit = 25

func (idx *index) update(ctx context.Context) {
	// state := gtkcord.FromContext(ctx).Offline()
	items := make([]indexItem, 0, 250)

	// dms, err := state.PrivateChannels()
	// if err != nil {
	// 	app.Error(ctx, err)
	// 	return
	// }

	// for i := range dms {
	// 	items = append(items, newChannelItem(state, nil, &dms[i]))
	// }

	// guilds, err := state.Guilds()
	// if err != nil {
	// 	app.Error(ctx, err)
	// 	return
	// }

	// for i, guild := range guilds {
	// 	chs, err := state.Channels(guild.ID, gtkcord.AllowedChannelTypes)
	// 	if err != nil {
	// 		log.Print("quickswitcher: cannot populate channels for guild ", guild.Name, ": ", err)
	// 		continue
	// 	}
	// 	items = append(items, newGuildItem(&guilds[i]))
	// 	for j := range chs {
	// 		items = append(items, newChannelItem(state, &guilds[i], &chs[j]))
	// 	}
	// }

	idx.items = items
}

func (idx *index) search(str string) []indexItem {
	if idx.items == nil {
		return nil
	}

	idx.buffer = idx.buffer[:0]
	if idx.buffer == nil {
		idx.buffer = make([]indexItem, 0, searchLimit)
	}

	matches := fuzzy.FindFrom(str, idx.items)
	for i := 0; i < len(matches) && i < searchLimit; i++ {
		idx.buffer = append(idx.buffer, idx.items[matches[i].Index])
	}

	return idx.buffer
}
