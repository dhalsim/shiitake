package global

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"slices"
	"sync"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip29"
	"github.com/puzpuzpuz/xsync/v3"
)

var groups = xsync.NewMapOf[string, *Group]()

type Group struct {
	nip29.Group
	Messages     []*nostr.Event
	NewMessage   chan *nostr.Event
	GroupUpdated chan struct{}
	NewError     chan error
	EOSE         chan struct{}
}

var getGroupMutex sync.Mutex

func GetGroup(ctx context.Context, gad nip29.GroupAddress) *Group {
	getGroupMutex.Lock()
	defer getGroupMutex.Unlock()

	if group, ok := groups.Load(gad.String()); ok {
		return group
	}

	group := &Group{
		Group: nip29.Group{
			Address: gad,
			Name:    gad.ID,
			Members: make(map[string]*nip29.Role, 5),
		},
		Messages:     make([]*nostr.Event, 0, 500),
		GroupUpdated: make(chan struct{}),
		NewMessage:   make(chan *nostr.Event),
		NewError:     make(chan error),
		EOSE:         make(chan struct{}),
	}
	groups.Store(gad.String(), group)

	relay, err := sys.Pool.EnsureRelay(group.Address.Relay)
	if err != nil {
		group.NewError <- fmt.Errorf("connect error: %w", err)
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
		group.NewError <- fmt.Errorf("subscription error: %w", err)
		return group
	}

	go func() {
		log.Printf("opening subscription to %s", group.Address)
		eosed := false
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
					group.GroupUpdated <- struct{}{}
				case 39001:
					group.Group.MergeInAdminsEvent(evt)
					group.GroupUpdated <- struct{}{}
				case 39002:
					group.Group.MergeInMembersEvent(evt)
					group.GroupUpdated <- struct{}{}
				case 9, 10:
					if eosed {
						group.NewMessage <- evt
						group.Messages = append(group.Messages, evt)
					} else {
						group.Messages = append(group.Messages, evt)
					}
				}
			case <-sub.EndOfStoredEvents:
				slices.SortFunc(group.Messages, func(a, b *nostr.Event) int {
					return int(a.CreatedAt - b.CreatedAt)
				})
				eosed = true
				close(group.EOSE)
			case <-ctx.Done():
				// when we leave a group or when we were just browsing it and leave, we close the subscription
				// and remove it from our list of cached groups
				getGroupMutex.Lock()
				groups.Delete(gad.String())
				getGroupMutex.Unlock()
				return
			}
		}
	}()

	return group
}

func JoinGroup(ctx context.Context, gad nip29.GroupAddress) {
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
