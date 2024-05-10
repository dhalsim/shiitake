package global

import (
	"context"
	"log"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip29"
	"github.com/puzpuzpuz/xsync/v3"
)

var groups = xsync.NewMapOf[string, *Group]()

func GetGroup(ctx context.Context, gad nip29.GroupAddress) *Group {
	group := &Group{
		Group: nip29.Group{
			Address: gad,
			Name:    gad.ID,
			Members: make(map[string]*nip29.Role, 5),
		},
		Messages: make([]*nostr.Event, 0, 500),
	}

	if err := subscribeGroup(ctx, group); err != nil {
		return nil
	}

	groups.Store(group.Address.ID, group)
	return group
}

func JoinGroup(ctx context.Context, gad nip29.GroupAddress) {
}

type Group struct {
	nip29.Group
	Messages []*nostr.Event
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
