package global

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/bep/debounce"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip29"
)

var groups = make(map[string]*Group)

type Group struct {
	nip29.Group
	NewMessage chan *nostr.Event

	update struct {
		listeners []func()
		debouncer func(func())
	}
}

var getGroupMutex sync.Mutex

func GetGroup(ctx context.Context, gad nip29.GroupAddress) *Group {
	getGroupMutex.Lock()
	defer getGroupMutex.Unlock()

	if group, ok := groups[gad.String()]; ok {
		return group
	}

	group := &Group{
		Group: nip29.Group{
			Address: gad,
			Name:    gad.ID,
			Members: make(map[string]*nip29.Role, 5),
		},
		NewMessage: make(chan *nostr.Event),
	}
	groups[gad.String()] = group

	relay, err := sys.Pool.EnsureRelay(group.Address.Relay)
	if err != nil {
		slog.Warn("connect error", "relay", group.Address.Relay, "err", err)
		return group
	}

	sub, err := relay.Subscribe(ctx, nostr.Filters{
		{
			Kinds: []int{39000, 39001, 39002},
			Tags: nostr.TagMap{
				"d": []string{group.Address.ID},
			},
			Limit: 3,
		},
		{
			Kinds: []int{9, 10},
			Tags: nostr.TagMap{
				"h": []string{group.Address.ID},
			},
			Limit: 500,
		},
	})
	if err != nil {
		slog.Warn("subscription error", "relay", group.Address.Relay, "err", err)
		return group
	}

	group.update.debouncer = debounce.New(700 * time.Millisecond)

	go func() {
		log.Printf("opening subscription to %s", group.Address)
		for {
			select {
			case evt, ok := <-sub.Events:
				if !ok {
					slog.Warn("subscription closed", "group", group.Address)
					return
				}

				switch evt.Kind {
				case 39000:
					group.Group.MergeInMetadataEvent(evt)
					for _, fn := range group.update.listeners {
						group.update.debouncer(fn)
					}
				case 39001:
					group.Group.MergeInAdminsEvent(evt)
					for _, fn := range group.update.listeners {
						group.update.debouncer(fn)
					}
				case 39002:
					group.Group.MergeInMembersEvent(evt)
					for _, fn := range group.update.listeners {
						group.update.debouncer(fn)
					}
				case 9, 10:
					group.NewMessage <- evt
				}
			case <-ctx.Done():
				// when we leave a group or when we were just browsing it and leave, we close the subscription
				// and remove it from our list of cached groups
				getGroupMutex.Lock()
				delete(groups, gad.String())
				getGroupMutex.Unlock()
				return
			}
		}
	}()

	return group
}

func (g *Group) OnUpdated(fn func()) { g.update.listeners = append(g.update.listeners, fn) }

func JoinGroup(ctx context.Context, gad nip29.GroupAddress) error {
	since := nostr.Now() - 1

	// ask to join group
	joinRequest := nostr.Event{
		Kind:      nostr.KindSimpleGroupJoinRequest,
		CreatedAt: nostr.Now(),
		Tags:      nostr.Tags{nostr.Tag{"h", gad.ID}},
	}
	if err := sys.Signer.SignEvent(&joinRequest); err != nil {
		return err
	}
	groupRelay, err := sys.Pool.EnsureRelay(gad.Relay)
	if err != nil {
		return err
	}
	if err := groupRelay.Publish(ctx, joinRequest); err != nil {
		return err
	}

	// wait for an automatic response -- or nothing
	sctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	sub, err := groupRelay.Subscribe(sctx, nostr.Filters{
		{
			Kinds: []int{nostr.KindSimpleGroupAddUser},
			Tags: nostr.TagMap{
				"h": []string{gad.ID},
				"p": []string{joinRequest.PubKey},
			},
			Since: &since,
		},
	})
	if err != nil {
		return err
	}

	select {
	case <-sub.Events:
		// if an event comes that means we are successful and will move on to the next step
	case <-sctx.Done():
		// otherwise we were denied
		return fmt.Errorf("not authorized to join group")
	}

	// add group to user list of groups
	newTag := []string{"group", gad.ID, gad.Relay}
	var found *nostr.Tag
	if me.lastList == nil {
		me.lastList = &nostr.Event{
			CreatedAt: nostr.Now(),
			Kind:      10009,
			Tags:      make(nostr.Tags, 0, 1),
		}
	} else {
		found = me.lastList.Tags.GetFirst(newTag)
	}
	if found == nil {
		// this is new, add
		me.lastList.Tags = append(me.lastList.Tags, newTag)
	}
	me.lastList.CreatedAt = nostr.Now()
	for _, url := range sys.FetchOutboxRelays(ctx, me.PubKey) {
		relay, _ := sys.Pool.EnsureRelay(url)

		if err := sys.Signer.SignEvent(me.lastList); err != nil {
			panic(err)
		}
		relay.Publish(ctx, *me.lastList)
	}

	return nil
}

func LeaveGroup(ctx context.Context, gad nip29.GroupAddress) {
	if me.lastList != nil {
		return
	}
	before := len(me.lastList.Tags)
	me.lastList.Tags = slices.DeleteFunc(me.lastList.Tags, func(t nostr.Tag) bool {
		return t[0] == "group" &&
			t[1] == gad.ID &&
			t[2] == gad.Relay
	})
	after := len(me.lastList.Tags)

	if before != after {
		// we have removed something, update
		me.lastList.CreatedAt = nostr.Now()
		for _, url := range sys.FetchOutboxRelays(ctx, me.PubKey) {
			relay, _ := sys.Pool.EnsureRelay(url)

			if err := sys.Signer.SignEvent(me.lastList); err != nil {
				panic(err)
			}
			relay.Publish(ctx, *me.lastList)
		}
	}
}

func (g Group) SendChatMessage(ctx context.Context, text string, replyTo string) error {
	evt := nostr.Event{
		Kind: 9,
		Tags: nostr.Tags{
			nostr.Tag{"h", g.Address.ID},
		},
		CreatedAt: nostr.Now(),
		Content:   text,
	}
	if replyTo != "" {
		evt.Tags = append(evt.Tags, nostr.Tag{"e", replyTo})
	}

	if err := sys.Signer.SignEvent(&evt); err != nil {
		return fmt.Errorf("failed to sign: %w", err)
	}

	relay, err := sys.Pool.EnsureRelay(g.Address.Relay)
	if err != nil {
		return fmt.Errorf("connection to '%s' failed: %w", g.Address.Relay, err)
	}

	if err := relay.Publish(ctx, evt); err != nil {
		return fmt.Errorf("publish to %s failed: %w", g.Address, err)
	}

	return nil
}
