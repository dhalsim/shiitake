package main

import (
	"context"
	"html"
	"log/slog"
	"strconv"
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/core/gioutil"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/app/locale"
	"github.com/diamondburned/gotkit/gtkutil"
	"github.com/nbd-wtf/go-nostr/nip29"
)

type messageReaction struct {
	Count        int
	Emoji        string
	GroupAddress nip29.GroupAddress
	MessageID    string
	Me           bool
}

type contentReactions struct {
	*gtk.FlowBox

	// *gtk.ScrolledWindow
	// grid *gtk.GridView

	ctx       context.Context
	parent    *Content
	reactions *gioutil.ListModel[messageReaction]
}

func newContentReactions(ctx context.Context, parent *Content) *contentReactions {
	rs := contentReactions{
		ctx:       ctx,
		parent:    parent,
		reactions: gioutil.NewListModel[messageReaction](),
	}

	// TODO: complain to the GTK devs about how broken GridView is.
	// Why is it not reflowing widgets? and other mysteries to solve in the GTK
	// framework.

	// rs.grid = gtk.NewGridView(
	// 	gtk.NewNoSelection(rs.reactions.ListModel),
	// 	newContentReactionsFactory(ctx))
	// rs.grid.SetOrientation(gtk.OrientationHorizontal)
	// reactionsCSS(rs.grid)
	//
	// rs.ScrolledWindow = gtk.NewScrolledWindow()
	// rs.ScrolledWindow.SetPolicy(gtk.PolicyNever, gtk.PolicyNever)
	// rs.ScrolledWindow.SetPropagateNaturalWidth(true)
	// rs.ScrolledWindow.SetPropagateNaturalHeight(false)
	// rs.ScrolledWindow.SetChild(rs.grid)

	rs.FlowBox = gtk.NewFlowBox()
	rs.FlowBox.SetOrientation(gtk.OrientationHorizontal)
	rs.FlowBox.SetHomogeneous(true)
	rs.FlowBox.SetMaxChildrenPerLine(30)
	rs.FlowBox.SetSelectionMode(gtk.SelectionNone)

	rs.FlowBox.BindModel(rs.reactions.ListModel, func(o *glib.Object) gtk.Widgetter {
		reaction := gioutil.ObjectValue[messageReaction](o)
		w := newContentReaction()
		w.SetReaction(ctx, rs.FlowBox, reaction)
		return w
	})

	gtkutil.BindActionCallbackMap(rs, map[string]gtkutil.ActionCallback{
		"reactions.toggle": {
			ArgType: glib.NewVariantType("s"),
			Func: func(args *glib.Variant) {
				// 		emoji := discord.APIEmoji(args.String())
				// 			selected := rs.isReacted(emoji)

				// client := gtkcord.FromContext(rs.ctx).Online()
				// gtkutil.Async(rs.ctx, func() func() {
				// 	var err error
				// 	if selected {
				// 		err = client.Unreact(rs.parent.ChannelID(), rs.parent.MessageID(), emoji)
				// 	} else {
				// 		err = client.React(rs.parent.ChannelID(), rs.parent.MessageID(), emoji)
				// 	}

				// 	if err != nil {
				// 		if selected {
				// 			err = fmt.Errorf("failed to react: %w", err)
				// 		} else {
				// 			err = fmt.Errorf("failed to unreact: %w", err)
				// 		}
				// 		app.Error(rs.ctx, err)
				// 	}

				// 	return nil
				// })
			},
		},
	})

	return &rs
}

func (rs *contentReactions) findReactionIx(emoji string) int {
	var i int
	foundIx := -1

	iter := rs.reactions.AllItems()
	iter(func(reaction messageReaction) bool {
		if reaction.Emoji == emoji {
			foundIx = i
			return false
		}
		i++
		return true
	})

	return foundIx
}

func (rs *contentReactions) isReacted(emoji string) bool {
	ix := rs.findReactionIx(emoji)
	if ix == -1 {
		return false
	}
	return rs.reactions.Item(ix).Me
}

// SetReactions sets the reactions of the message.
//
// TODO: implement Add and Remove event handlers directly in this container to
// avoid having to clear the whole list.
func (rs *contentReactions) SetReactions(reactions []string) {
	messageReactions := make([]messageReaction, len(reactions))
	for i, r := range reactions {
		messageReactions[i] = messageReaction{
			Emoji:        r,
			GroupAddress: win.main.Messages.currentGroup.Address,
			MessageID:    rs.parent.MessageID,
		}
	}
	rs.reactions.Splice(0, rs.reactions.NItems(), messageReactions...)
}

/*
func newContentReactionsFactory(ctx context.Context) *gtk.ListItemFactory {
	reactionWidgets := make(map[uintptr]*contentReaction)

	factory := gtk.NewSignalListItemFactory()
	factory.ConnectSetup(func(item *gtk.ListItem) {
		w := newContentReaction()
		item.SetChild(w)
		reactionWidgets[item.Native()] = w
	})
	factory.ConnectTeardown(func(item *gtk.ListItem) {
		item.SetChild(nil)
		delete(reactionWidgets, item.Native())
	})

	factory.ConnectBind(func(item *gtk.ListItem) {
		reaction := gioutil.ObjectValue[messageReaction](item.Item())

		w := reactionWidgets[item.Native()]
		w.SetReaction(ctx, reaction)
	})
	factory.ConnectUnbind(func(item *gtk.ListItem) {
		w := reactionWidgets[item.Native()]
		w.Clear()
	})

	return &factory.ListItemFactory
}
*/

type reactionsLoadState uint8

const (
	reactionsNotLoaded reactionsLoadState = iota
	reactionsLoading
	reactionsLoaded
)

type contentReaction struct {
	*gtk.ToggleButton
	iconBin    *adw.Bin
	countLabel *gtk.Label

	reaction messageReaction

	tooltip      string
	tooltipState reactionsLoadState
}

func newContentReaction() *contentReaction {
	r := contentReaction{}

	r.ToggleButton = gtk.NewToggleButton()
	r.ToggleButton.ConnectClicked(func() {
		r.SetSensitive(false)

		ok := r.ActivateAction("reactions.toggle", glib.NewVariantString(string(r.reaction.Emoji)))
		if !ok {
			slog.Error(
				"failed to activate reactions.toggle",
				"emoji", r.reaction.Emoji)
		}
	})

	r.ToggleButton.SetHasTooltip(true)
	r.ToggleButton.ConnectQueryTooltip(func(_, _ int, _ bool, tooltip *gtk.Tooltip) bool {
		tooltip.SetText(locale.Get("Loading..."))
		r.invalidateUsers(tooltip.SetMarkup)
		return true
	})

	r.iconBin = adw.NewBin()

	r.countLabel = gtk.NewLabel("")
	r.countLabel.SetHExpand(true)
	r.countLabel.SetXAlign(1)

	box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	box.Append(r.iconBin)
	box.Append(r.countLabel)

	r.ToggleButton.SetChild(box)

	return &r
}

// SetReaction sets the reaction of the widget.
func (r *contentReaction) SetReaction(ctx context.Context, flowBox *gtk.FlowBox, reaction messageReaction) {
	r.reaction = reaction
	// r.client = gtkcord.FromContext(ctx).Online()

	if strings.HasPrefix(reaction.Emoji, ":") {
		// emoji := avatar.New(ctx, imgutil.HTTPProvider)
		// emoji.SetSizeRequest(13, 13)
		// emoji.SetKeepAspectRatio(true)
		// emoji.SetURL(reaction.Emoji)

		// TODO: get this working:
		// Currently, it just jitters in size. The button itself can still be
		// sized small, FlowBox is just forcing it to be big. This does mean
		// that it's not the GIF that is causing this.

		// anim := emoji.EnableAnimation()
		// anim.ConnectMotion(r)

		// r.iconBin.SetChild(emoji)
	} else {
		label := gtk.NewLabel(reaction.Emoji)

		r.iconBin.SetChild(label)
	}

	r.countLabel.SetLabel(strconv.Itoa(reaction.Count))

	r.ToggleButton.SetActive(reaction.Me)
	if reaction.Me {
	} else {
	}
}

func (r *contentReaction) Clear() {
	r.reaction = messageReaction{}
	r.tooltipState = reactionsNotLoaded
	r.iconBin.SetChild(nil)
	r.ToggleButton.SetActive(false)
	r.ToggleButton.RemoveCSSClass("message-reaction-me")
}

func (r *contentReaction) invalidateUsers(callback func(string)) {
	if r.tooltipState != reactionsNotLoaded {
		callback(r.tooltip)
		return
	}

	r.tooltipState = reactionsLoading
	r.tooltip = ""

	reaction := r.reaction

	var tooltip string
	if strings.HasPrefix(reaction.Emoji, ":") {
		tooltip = html.EscapeString(reaction.Emoji) + "\n"
	}

	done := func(tooltip string, err error) {
		glib.IdleAdd(func() {
			if err != nil {
				r.tooltipState = reactionsNotLoaded
				r.tooltip = tooltip + "<b>" + locale.Get("Error: ") + "</b>" + err.Error()

				slog.Error(
					"cannot load reaction tooltip",
					"channel", reaction.GroupAddress.ID,
					"message", reaction.MessageID,
					"emoji", reaction.Emoji,
					"err", err)
			} else {
				r.tooltipState = reactionsLoaded
				r.tooltip = tooltip
			}

			callback(r.tooltip)
		})
	}

	go func() {
		// u, err := client.Reactions(
		// 	reaction.ChannelID,
		// 	reaction.MessageID,
		// 	reaction.Emoji.APIString(), 11)
		// if err != nil {
		// 	done(tooltip, err)
		// 	return
		// }

		// var hasMore bool
		// if len(u) > 10 {
		// 	hasMore = true
		// 	u = u[:10]
		// }

		// for _, user := range u {
		// 	tooltip += fmt.Sprintf(
		// 		`<span size="small">%s</span>`+"\n",
		// 		client.MemberMarkup(reaction.GuildID, &discord.GuildUser{User: user}),
		// 	)
		// }

		// if hasMore {
		// 	tooltip += "..."
		// } else {
		// 	tooltip = strings.TrimRight(tooltip, "\n")
		// }

		done(tooltip, nil)
	}()
}
