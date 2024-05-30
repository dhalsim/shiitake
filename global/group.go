package global

import (
	"context"
	"fmt"
	"log"
	"slices"

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

func GetGroup(ctx context.Context, gad nip29.GroupAddress) *Group {
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
				fmt.Println("event", evt)

				if !ok {
					log.Printf("subscription to %s closed", group.Address)
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
		for _, url := range sys.FetchOutboxRelays(ctx, me.PubKey) {
			relay, _ := sys.Pool.EnsureRelay(url)

			if err := sys.Signer.SignEvent(me.lastList); err != nil {
				panic(err)
			}
			relay.Publish(ctx, *me.lastList)
		}
	}
}

func (g Group) SendChatMessage(ctx context.Context, text string) {
	evt := nostr.Event{
		Kind: 9,
		Tags: nostr.Tags{
			nostr.Tag{"h", g.Address.ID},
		},
		CreatedAt: nostr.Now(),
		Content:   text,
	}
	if err := sys.Signer.SignEvent(&evt); err != nil {
		panic(err)
	}

	relay, err := sys.Pool.EnsureRelay(g.Address.Relay)
	if err != nil {
		log.Printf("failed to connect to relay '%s': %s\n", g.Address.Relay, err)
		return
	}

	if err := relay.Publish(ctx, evt); err != nil {
		log.Println("failed to publish message: ", err)
		return
	}
}

func (g Group) SubscribeToMessages(ctx context.Context) {
	relay, err := sys.Pool.EnsureRelay(g.Address.Relay)
	if err != nil {
		log.Printf("failed to connect to relay '%s': %s\n", g.Address.Relay, err)
		return
	}

	sub, err := relay.Subscribe(ctx, nostr.Filters{
		{
			Kinds: []int{9},
			Tags: nostr.TagMap{
				"h": []string{g.Address.ID},
			},
			Limit: 500,
		},
	})
	if err != nil {
		log.Printf("failed to subscribe to group %s: %s\n", g.Address, err)
		return
	}

	{
		stored := make([]*nostr.Event, 0, 500)
		for {
			select {
			case evt := <-sub.Events:
				// send stored messages in a big batch first
				stored = append(stored, evt)
			case <-sub.EndOfStoredEvents:
				// reverse and send
				n := len(stored)
				log.Println("received stored", n)
				goto continuous
			}
		}
	}

continuous:
	// after we got an eose we will just send messages as they come one by one
	for evt := range sub.Events {
		log.Println(evt)
	}
}
