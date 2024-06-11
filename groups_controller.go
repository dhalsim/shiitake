package main

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"fiatjaf.com/shiitake/components/icon_placeholder"
	"fiatjaf.com/shiitake/global"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotkit/app"
	"github.com/nbd-wtf/go-nostr/nip19"
	"github.com/nbd-wtf/go-nostr/nip29"
)

type GroupsController struct {
	*gtk.Stack

	LoadMore *gtk.Button
	Composer *ComposerView

	placeholder gtk.Widgetter

	switching sync.Mutex
	groups    map[string]*GroupView
	current   *GroupView

	ctx context.Context
}

func NewGroupsController(ctx context.Context) *GroupsController {
	v := &GroupsController{
		ctx:    ctx,
		groups: make(map[string]*GroupView),
	}

	v.Stack = gtk.NewStack()
	v.Stack.SetTransitionType(gtk.StackTransitionTypeCrossfade)
	v.Stack.SetHExpand(true)
	v.Stack.SetVExpand(true)

	plc := icon_placeholder.New("chat-bubbles-empty-symbolic")
	v.Stack.AddNamed(plc, "placeholder")

	// if the window gains focus, try to carefully mark the channel as read.
	var windowSignal glib.SignalHandle
	v.ConnectMap(func() {
		window := app.GTKWindowFromContext(ctx)
		windowSignal = window.NotifyProperty("is-active", func() {
			if v.IsActive() {
				// 		v.MarkRead()
			}
		})
	})
	// Immediately disconnect the signal when the widget is unmapped.
	// This should prevent v from being referenced forever.
	v.ConnectUnmap(func() {
		window := app.GTKWindowFromContext(ctx)
		window.HandlerDisconnect(windowSignal)
		windowSignal = 0
	})

	return v
}

func (v *GroupsController) switchTo(gad nip29.GroupAddress) {
	if !gad.IsValid() {
		// empty, switch to placeholder
		v.Stack.SetVisibleChildName("placeholder")
		return
	}

	v.switching.Lock()
	defer v.switching.Unlock()

	if v.current != nil && v.current.group.Address.Equals(gad) {
		return
	}
	group := global.GetGroup(v.ctx, gad)

	win.main.Header.SetTitleWidget(adw.NewWindowTitle(group.Name, group.Address.String()))

	// get existing group view
	groupView, ok := v.groups[gad.String()]
	if !ok {
		// create
		groupView = NewGroupView(v.ctx, group)

		// insert in the stack and keep track of this
		v.Stack.AddNamed(groupView, gad.String())
		v.groups[gad.String()] = groupView

		// set to update window header title
		groupView.group.OnUpdated(func() {
			if v.current == groupView {
				win.main.Header.SetTitleWidget(adw.NewWindowTitle(group.Name, group.Address.String()))
			}
		})
	}

	v.current = groupView

	// make it visible
	v.Stack.SetVisibleChild(groupView)
}

func (v *GroupsController) currentGroup() *global.Group {
	v.switching.Lock()
	defer v.switching.Unlock()
	if v.current == nil {
		return nil
	}
	return v.current.group
}

func (v *GroupsController) updateMember(list *gtk.ListBox, pubkey string) {
	eachChild(list, func(lbr *gtk.ListBoxRow) bool {
		fmt.Println("updating member", pubkey)

		// fragile: this depends on the hierarchy of components: message > rightBox > topLabel
		label := lbr.Child().(*gtk.Box).LastChild().(*gtk.Box).FirstChild().(*gtk.Label)
		// fragile: this depends on the string given to the tooltip
		npub := strings.Split(strings.Split(label.TooltipMarkup(), "(")[1], ")")[0]
		_, data, _ := nip19.Decode(npub)
		if pubkey == data.(string) {
			// replace avatar
			// avatar := lbr.Child().(*gtk.Box).FirstChild()
			// lbr.Child().(*gtk.Box).InsertBefore(newAvatar, avatar)
			// lbr.Child().(*gtk.Box).Remove(avatar)

			// replace toplabel
			// TODO
		}
		return false
	})
}

// IsActive returns true if GroupsController is active and visible. This implies that the
// window is focused.
func (v *GroupsController) IsActive() bool {
	win := app.GTKWindowFromContext(v.ctx)
	return win.IsActive() && v.Mapped()
}
