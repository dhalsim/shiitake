package global

import (
	"context"
	"log"

	"github.com/nbd-wtf/go-nostr"
)

func GetChannel(ctx context.Context, id string) Channel {
	return Channel{id, ""}
}

type Channel struct {
	ID       string
	RelayURL string
}

func (channel Channel) SendChatMessage(ctx context.Context, text string) {
	evt := nostr.Event{
		Kind: 9,
		Tags: nostr.Tags{
			nostr.Tag{"h", channel.ID},
		},
		CreatedAt: nostr.Now(),
		Content:   text,
	}
	if err := Sys.Signer.SignEvent(&evt); err != nil {
		panic(err)
	}

	relay, err := Sys.Pool.EnsureRelay(channel.RelayURL)
	if err != nil {
		log.Printf("failed to connect to relay '%s': %s\n", channel.RelayURL, err)
		return
	}

	if err := relay.Publish(ctx, evt); err != nil {
		log.Println("failed to publish message: ", err)
		return
	}
}

func (channel Channel) SubscribeToMessages(ctx context.Context) {
	relay, err := Sys.Pool.EnsureRelay(channel.RelayURL)
	if err != nil {
		log.Printf("failed to connect to relay '%s': %s\n", channel.RelayURL, err)
		return
	}

	sub, err := relay.Subscribe(ctx, nostr.Filters{
		{
			Kinds: []int{9},
			Tags: nostr.TagMap{
				"h": []string{channel.ID},
			},
			Limit: 500,
		},
	})
	if err != nil {
		log.Printf("failed to subscribe to group %s at '%s': %s\n", channel.ID, channel.RelayURL, err)
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
