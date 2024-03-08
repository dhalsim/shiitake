package global

import (
	"context"
	"log"

	"github.com/nbd-wtf/go-nostr"
)

var mainStream = make(chan *nostr.Event)

func Start(ctx context.Context) {
	me := GetMe(ctx)

	go func() {
		for _, group := range me.Groups {
			go subscribeGroup(ctx, group)
		}
	}()

	for event := range mainStream {
		log.Println("got event", event)
	}
}

func subscribeGroup(ctx context.Context, group Group) {
	relay, err := sys.Pool.EnsureRelay(group.RelayURL)
	if err != nil {
		log.Printf("failed to connect to %s\n", group.RelayURL)
		return
	}

	sub, err := relay.Subscribe(ctx, nostr.Filters{
		{
			Kinds: []int{39000},
			Tags: nostr.TagMap{
				"d": []string{group.ID},
			},
		},
		{
			Kinds: []int{9},
			Tags: nostr.TagMap{
				"h": []string{group.ID},
			},
		},
	})
	if err != nil {
		log.Printf("failed to subscribe to %s\n", group.RelayURL)
		return
	}

	for {
		select {
		case evt, ok := <-sub.Events:
			if !ok {
				log.Printf("subscription to %s closed", group.RelayURL)
				return
			}
			mainStream <- evt
		}
	}
}
