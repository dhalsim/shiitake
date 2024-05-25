package window

import (
	"context"

	"fiatjaf.com/shiitake/global"
	"fiatjaf.com/shiitake/utils"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
	"github.com/nbd-wtf/go-nostr"
)

// ViewChild is a child inside the relays view. It is either a *Relay or a
// *Folder containing more *Relays.
type ViewChild interface {
	gtk.Widgetter
	viewChild()
}

// View contains a list of relays and folders.
type RelaysView struct {
	*gtk.Box
	Children []ViewChild

	current currentRelay
	ctx     context.Context
}

var relaysViewCSS = cssutil.Applier("relay-view", `
	.relay-view {
		margin: 4px 0;
	}
	.relay-view button:active:not(:hover) {
		background: initial;
	}
`)

func NewRelaysView(ctx context.Context) *RelaysView {
	v := RelaysView{
		ctx: ctx,
	}

	v.Box = gtk.NewBox(gtk.OrientationVertical, 0)
	viewCSS(v)

	go func() {
		me := global.GetMe(ctx)
		for {
			select {
			case relay := <-me.JoinedRelay:
				g := NewRelay(v.ctx, relay)
				v.prepend(g)
			case url := <-me.LeftRelay:
				relay := v.get(url)
				if relay != nil {
					v.remove(relay)
				}
			}
		}
	}()

	// cancellable := gtkutil.WithVisibility(ctx, v)

	// state := gtkcord.FromContext(ctx)
	// state.BindHandler(cancellable, func(ev gateway.Event) {
	// 	switch ev := ev.(type) {
	// 	case *gateway.ReadyEvent, *gateway.ResumedEvent:
	// 		// Recreate the whole list in case we have some new info.
	// 		v.Invalidate()

	// 	case *read.UpdateEvent:
	// 		if relay := v.Relay(ev.RelayID); relay != nil {
	// 			relay.InvalidateUnread()
	// 		}
	// 	case *gateway.GroupCreateEvent:
	// 		if ev.RelayID.IsValid() {
	// 			if relay := v.Relay(ev.RelayID); relay != nil {
	// 				relay.InvalidateUnread()
	// 			}
	// 		}
	// 	case *gateway.RelayCreateEvent:
	// 		if relay := v.Relay(ev.ID); relay != nil {
	// 			relay.Update(&ev.Relay)
	// 		} else {
	// 			v.AddRelay(&ev.Relay)
	// 		}
	// 	case *gateway.RelayUpdateEvent:
	// 		if relay := v.Relay(ev.ID); relay != nil {
	// 			relay.Invalidate()
	// 		}
	// 	case *gateway.RelayDeleteEvent:
	// 		if ev.Unavailable {
	// 			if relay := v.Relay(ev.ID); relay != nil {
	// 				relay.SetUnavailable()

	// 				parent := gtk.BaseWidget(relay.Parent())
	// 				parent.ActivateAction("win.reset-view", nil)
	// 				return
	// 			}
	// 		}

	// 		relay := v.RemoveRelay(ev.ID)
	// 		if relay != nil && relay.IsSelected() {
	// 			parent := gtk.BaseWidget(relay.Parent())
	// 			parent.ActivateAction("win.reset-view", nil)
	// 		}
	// 	}
	// })

	return &v
}

// Invalidate invalidates the view and recreates everything. Use with care.
func (v *RelaysView) Invalidate() {
	// TODO: reselect.

	// state := gtkcord.FromContext(v.ctx)
	// ready := state.Ready()

	// if ready.UserSettings != nil {
	// 	switch {
	// 	case ready.UserSettings.RelayPositions != nil:
	// 		v.SetRelaysFromIDs(ready.UserSettings.RelayPositions)
	// 	}
	// }

	// relays, err := state.Cabinet.Relays()
	// if err != nil {
	// 	app.Error(v.ctx, errors.Wrap(err, "cannot get relays"))
	// 	return
	// }

	// // Sort so that the relays that we've joined last are at the bottom.
	// // This means we can prepend relays as we go, and the latest one will be
	// // prepended to the top.
	// sort.Slice(relays, func(i, j int) bool {
	// 	ti, ok := state.RelayState.JoinedAt(relays[i].ID)
	// 	if !ok {
	// 		return false // put last
	// 	}
	// 	tj, ok := state.RelayState.JoinedAt(relays[j].ID)
	// 	if !ok {
	// 		return true
	// 	}
	// 	return ti.Before(tj)
	// })

	// // Construct a map of shownRelays relays, so we know to not create a
	// // relay if it's already shown.
	// shownRelays := make(map[string]struct{}, 200)
	// v.eachRelay(func(g *Relay) bool {
	// 	shownRelays[g.ID()] = struct{}{}
	// 	return false
	// })

	// for i, relay := range relays {
	// 	_, shown := shownRelays[relay.ID]
	// 	if shown {
	// 		continue
	// 	}

	// 	g := NewRelay(v.ctx, relay.ID)
	// 	g.Update(&relays[i])

	// 	// Prepend the relay.
	// 	v.prepend(g)
	// }
}

func (v *RelaysView) append(this ViewChild) {
	v.Children = append(v.Children, this)
	v.Box.Append(this)
}

func (v *RelaysView) prepend(this ViewChild) {
	v.Children = append(v.Children, nil)
	copy(v.Children[1:], v.Children)
	v.Children[0] = this

	v.Box.Prepend(this)
}

func (v *RelaysView) remove(this ViewChild) {
	for i, child := range v.Children {
		if child == this {
			v.Children = append(v.Children[:i], v.Children[i+1:]...)
			v.Box.Remove(child)
			break
		}
	}
}

func (v *RelaysView) clear() {
	for _, child := range v.Children {
		v.Box.Remove(child)
	}
	v.Children = nil
}

// SelectedRelayID returns the selected relay ID, if any.
func (v *RelaysView) SelectedRelayURL() string {
	if v.current.relay == nil {
		return ""
	}
	return v.current.relay.url
}

func (v *RelaysView) get(url string) *Relay {
	for _, child := range v.Children {
		switch child := child.(type) {
		case *Relay:
			if child.url == nostr.NormalizeURL(url) {
				return child
			}
		}
	}

	return nil
}

// saveSelection saves the current relay selection to be restored later using
// the returned callback.
func (v *RelaysView) saveSelection() (restore func()) {
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
