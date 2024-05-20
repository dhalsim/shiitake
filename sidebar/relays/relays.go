package relays

import (
	"context"
	"log"

	"fiatjaf.com/shiitake/utils"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
	"github.com/nbd-wtf/go-nostr"
)

// ViewChild is a child inside the guilds view. It is either a *Relay or a
// *Folder containing more *Relays.
type ViewChild interface {
	gtk.Widgetter
	viewChild()
}

// View contains a list of guilds and folders.
type View struct {
	*gtk.Box
	Children []ViewChild

	current currentRelay

	ctx context.Context
}

var viewCSS = cssutil.Applier("guild-view", `
	.guild-view {
		margin: 4px 0;
	}
	.guild-view button:active:not(:hover) {
		background: initial;
	}
`)

// NewView creates a new View.
func NewView(ctx context.Context) *View {
	v := View{
		ctx: ctx,
	}

	v.Box = gtk.NewBox(gtk.OrientationVertical, 0)
	viewCSS(v)

	// cancellable := gtkutil.WithVisibility(ctx, v)

	// state := gtkcord.FromContext(ctx)
	// state.BindHandler(cancellable, func(ev gateway.Event) {
	// 	switch ev := ev.(type) {
	// 	case *gateway.ReadyEvent, *gateway.ResumedEvent:
	// 		// Recreate the whole list in case we have some new info.
	// 		v.Invalidate()

	// 	case *read.UpdateEvent:
	// 		if guild := v.Relay(ev.RelayID); guild != nil {
	// 			guild.InvalidateUnread()
	// 		}
	// 	case *gateway.ChannelCreateEvent:
	// 		if ev.RelayID.IsValid() {
	// 			if guild := v.Relay(ev.RelayID); guild != nil {
	// 				guild.InvalidateUnread()
	// 			}
	// 		}
	// 	case *gateway.RelayCreateEvent:
	// 		if guild := v.Relay(ev.ID); guild != nil {
	// 			guild.Update(&ev.Relay)
	// 		} else {
	// 			v.AddRelay(&ev.Relay)
	// 		}
	// 	case *gateway.RelayUpdateEvent:
	// 		if guild := v.Relay(ev.ID); guild != nil {
	// 			guild.Invalidate()
	// 		}
	// 	case *gateway.RelayDeleteEvent:
	// 		if ev.Unavailable {
	// 			if guild := v.Relay(ev.ID); guild != nil {
	// 				guild.SetUnavailable()

	// 				parent := gtk.BaseWidget(guild.Parent())
	// 				parent.ActivateAction("win.reset-view", nil)
	// 				return
	// 			}
	// 		}

	// 		guild := v.RemoveRelay(ev.ID)
	// 		if guild != nil && guild.IsSelected() {
	// 			parent := gtk.BaseWidget(guild.Parent())
	// 			parent.ActivateAction("win.reset-view", nil)
	// 		}
	// 	}
	// })

	return &v
}

// Invalidate invalidates the view and recreates everything. Use with care.
func (v *View) Invalidate() {
	// TODO: reselect.

	// state := gtkcord.FromContext(v.ctx)
	// ready := state.Ready()

	// if ready.UserSettings != nil {
	// 	switch {
	// 	case ready.UserSettings.RelayPositions != nil:
	// 		v.SetRelaysFromIDs(ready.UserSettings.RelayPositions)
	// 	}
	// }

	// guilds, err := state.Cabinet.Relays()
	// if err != nil {
	// 	app.Error(v.ctx, errors.Wrap(err, "cannot get guilds"))
	// 	return
	// }

	// // Sort so that the guilds that we've joined last are at the bottom.
	// // This means we can prepend guilds as we go, and the latest one will be
	// // prepended to the top.
	// sort.Slice(guilds, func(i, j int) bool {
	// 	ti, ok := state.RelayState.JoinedAt(guilds[i].ID)
	// 	if !ok {
	// 		return false // put last
	// 	}
	// 	tj, ok := state.RelayState.JoinedAt(guilds[j].ID)
	// 	if !ok {
	// 		return true
	// 	}
	// 	return ti.Before(tj)
	// })

	// // Construct a map of shownRelays guilds, so we know to not create a
	// // guild if it's already shown.
	// shownRelays := make(map[string]struct{}, 200)
	// v.eachRelay(func(g *Relay) bool {
	// 	shownRelays[g.ID()] = struct{}{}
	// 	return false
	// })

	// for i, guild := range guilds {
	// 	_, shown := shownRelays[guild.ID]
	// 	if shown {
	// 		continue
	// 	}

	// 	g := NewRelay(v.ctx, guild.ID)
	// 	g.Update(&guilds[i])

	// 	// Prepend the guild.
	// 	v.prepend(g)
	// }
}

// AddRelay prepends a single guild into the view.
// func (v *View) AddRelay(guild *discord.Relay) {
// 	g := NewRelay(v.ctx, guild.ID)
// 	g.Update(guild)
//
// 	v.Box.Prepend(g)
// 	v.Children = append([]ViewChild{g}, v.Children...)
// }

// RemoveRelay removes the given relay.
func (v *View) RemoveRelay(url string) *Relay {
	guild := v.Relay(url)
	if guild == nil {
		return nil
	}

	v.remove(guild)
	return guild
}

// SetRelaysFromIDs calls SetRelays with guilds fetched from the state by the
// given ID list.
func (v *View) SetRelaysFromIDs(guildIDs []string) {
	restore := v.saveSelection()
	defer restore()

	v.clear()

	for _, id := range guildIDs {
		g := NewRelay(v.ctx, id)
		g.Invalidate()

		v.append(g)
	}
}

// SetRelays sets the guilds shown.
func (v *View) SetRelays(urls []string) {
	restore := v.saveSelection()
	defer restore()

	v.clear()

	for i, url := range urls {
		g := NewRelay(v.ctx, url)
		g.Update(urls[i])

		v.append(g)
	}
}

func (v *View) append(this ViewChild) {
	v.Children = append(v.Children, this)
	v.Box.Append(this)
}

func (v *View) prepend(this ViewChild) {
	v.Children = append(v.Children, nil)
	copy(v.Children[1:], v.Children)
	v.Children[0] = this

	v.Box.Prepend(this)
}

func (v *View) remove(this ViewChild) {
	for i, child := range v.Children {
		if child == this {
			v.Children = append(v.Children[:i], v.Children[i+1:]...)
			v.Box.Remove(child)
			break
		}
	}
}

func (v *View) clear() {
	for _, child := range v.Children {
		v.Box.Remove(child)
	}
	v.Children = nil
}

// SelectedRelayID returns the selected guild ID, if any.
func (v *View) SelectedRelayURL() string {
	if v.current.relay == nil {
		return ""
	}
	return v.current.relay.url
}

// Relay finds a guild inside View by its ID.
func (v *View) Relay(url string) *Relay {
	var guild *Relay
	v.eachRelay(func(g *Relay) bool {
		if g.ID() == nostr.NormalizeURL(url) {
			guild = g
			return true
		}
		return false
	})
	return guild
}

func (v *View) eachRelay(f func(*Relay) (stop bool)) {
	for _, child := range v.Children {
		switch child := child.(type) {
		case *Relay:
			if f(child) {
				return
			}
		}
	}
}

// SetSelectedRelay sets the selected guild. It does not propagate the selection
// to the sidebar.
func (v *View) SetSelectedRelay(id string) {
	guild := v.Relay(id)
	if guild == nil {
		log.Printf("guilds.View: cannot select guild %d: not found", id)
		v.Unselect()
		return
	}

	current := currentRelay{
		relay: guild,
	}

	if current != v.current {
		v.Unselect()
		v.current = current
		v.current.SetSelected(true)
	}
}

// Unselect unselects any guilds inside this guild view. Use this when the
// window is showing a channel that's not from any guild.
func (v *View) Unselect() {
	v.current.Unselect()
	v.current = currentRelay{}
}

// saveSelection saves the current guild selection to be restored later using
// the returned callback.
func (v *View) saveSelection() (restore func()) {
	if v.current.relay == nil {
		// Nothing to restore.
		return func() {}
	}

	return func() {
		parent := gtk.BaseWidget(v.Parent())
		parent.ActivateAction("win.open-relay", utils.NewRelayURLVariant(v.current.relay.url))
	}
}

type currentRelay struct {
	relay *Relay
}

func (c currentRelay) Unselect() {
	c.SetSelected(false)
}

func (c currentRelay) SetSelected(selected bool) {
	if c.relay != nil {
		c.relay.SetSelected(selected)
	}
}
