package global

import (
	"context"
	"fmt"
	"log"

	"github.com/nbd-wtf/go-nostr"
)

func Start(ctx context.Context) {
	me := GetMe(ctx)

	for _, group := range me.Groups {
		err := subscribeGroup(ctx, group)
		if err != nil {
			log.Printf("failed to subscribe to %s: %s\n", group.Address, err)
		}
	}
}

func subscribeGroup(ctx context.Context, group *Group) error {
	relay, err := sys.Pool.EnsureRelay(group.Address.Relay)
	if err != nil {
		return fmt.Errorf("connect error: %w", err)
	}

	sub, err := relay.Subscribe(ctx, nostr.Filters{
		{
			Kinds: []int{39000, 39001, 39002},
			Tags: nostr.TagMap{
				"d": []string{group.Address.ID},
			},
			Limit: 500,
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
		return fmt.Errorf("subscription error: %w", err)
	}

	go func() {
		for {
			select {
			case evt, ok := <-sub.Events:
				if !ok {
					log.Printf("subscription to %s closed", group.Address)
				}

				switch evt.Kind {
				case 39000:
					group.Group.MergeInMetadataEvent(evt)
				case 39001:
					group.Group.MergeInAdminsEvent(evt)
				case 39002:
					group.Group.MergeInMembersEvent(evt)
				case 9, 10:
					group.Messages = append(group.Messages, evt)
				}
			}
		}
	}()

	return nil
}
