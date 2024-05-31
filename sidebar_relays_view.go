package main

import (
	"context"

	"fiatjaf.com/shiitake/global"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/gtkutil/cssutil"
)

// View contains a list of relays and folders.
type RelaysView struct {
	Widget *gtk.ListBox

	ctx context.Context
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

	v.Widget = gtk.NewListBox()
	relaysViewCSS(v.Widget)

	go func() {
		me := global.GetMe(ctx)
		for {
			select {
			case relay := <-me.JoinedRelay:
				g := NewRelay(v.ctx, relay)
				v.Widget.Prepend(g)
			case url := <-me.LeftRelay:
				row := getChild(v.Widget, func(lbr *gtk.ListBoxRow) bool { return url == lbr.Name() })
				if row != nil {
					v.Widget.Remove(row)
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
