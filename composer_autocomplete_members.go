package main

import (
	"context"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/chatkit/components/autocomplete"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/sahilm/fuzzy"
)

const memberCacheExpiry = 2 * time.Second

type members []discord.Member

func (m members) String(i int) string { return m[i].Nick + m[i].User.DisplayName + m[i].User.Tag() }
func (m members) Len() int            { return len(m) }

type memberCompleter struct {
	members members
	matched []autocomplete.Data
	updated time.Time
	guildID string
}

// NewMemberCompleter creates a new autocomplete searcher that searches for
// members.
func NewMemberCompleter(guildId string) autocomplete.Searcher {
	return &memberCompleter{
		guildID: guildId,
		matched: make([]autocomplete.Data, 0, 15),
	}
}

func (c *memberCompleter) Rune() rune { return '@' }

func (c *memberCompleter) Search(ctx context.Context, str string) []autocomplete.Data {
	if len(str) < 1 {
		return nil
	}

	// state := gtkcord.FromContext(ctx)
	if len(str) > 2 {
		// state.MemberState.SearchMember(c.guildID, str)
	}

	now := time.Now()

	c.updated = now
	if data := c.search(str); len(data) > 0 {
		return data
	}

	return nil
}

func (c *memberCompleter) search(str string) []autocomplete.Data {
	res := fuzzy.FindFrom(str, c.members)
	if len(res) > 15 {
		res = res[:15]
	}

	data := c.matched[:0]
	for _, r := range res {
		data = append(data, MemberData(c.members[r.Index]))
	}

	return data
}

// MemberData is the Data structure for each member.
type MemberData discord.Member

func (d MemberData) Row(ctx context.Context) *gtk.ListBoxRow {
	// i := onlineimage.NewAvatar(ctx, imgutil.HTTPProvider, emojiSize)
	// i.AddCSSClass("autocompleter-customemoji")
	// i.SetFromURL(gtkcord.InjectAvatarSize(d.User.AvatarURLWithType(discord.PNGImage)))

	// l := gtk.NewLabel("")
	// l.SetMaxWidthChars(45)
	// l.SetWrap(false)
	// l.SetEllipsize(pango.EllipsizeEnd)
	// l.SetXAlign(0)
	// l.SetJustify(gtk.JustifyLeft)

	// if d.Nick != "" {
	// 	l.SetLines(2)
	// 	l.SetMarkup(fmt.Sprintf(
	// 		`%s`+"\n"+`<span size="smaller" fgalpha="75%%" rise="-1200">%s</span>`,
	// 		html.EscapeString(d.Nick),
	// 		html.EscapeString(d.User.Tag()),
	// 	))
	// } else {
	// 	l.SetLines(1)
	// 	l.SetText(d.User.Tag())
	// }

	// b := gtk.NewBox(gtk.OrientationHorizontal, 4)
	// b.Append(i)
	// b.Append(l)

	r := gtk.NewListBoxRow()
	r.AddCSSClass("autocomplete-member")
	// r.SetChild(b)

	return r
}
