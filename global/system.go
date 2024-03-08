package global

import (
	"context"
	"time"

	"github.com/nbd-wtf/go-nostr"
	sdk "github.com/nbd-wtf/nostr-sdk"
	cache_memory "github.com/nbd-wtf/nostr-sdk/cache/memory"
)

var sys = &sdk.System{
	Pool:             nostr.NewSimplePool(context.Background()),
	RelaysCache:      cache_memory.New32[[]sdk.Relay](1000),
	MetadataCache:    cache_memory.New32[sdk.ProfileMetadata](1000),
	FollowsCache:     cache_memory.New32[[]sdk.Follow](1),
	RelayListRelays:  []string{"wss://purplepag.es", "wss://relay.nostr.band"},
	FollowListRelays: []string{"wss://public.relaying.io", "wss://relay.nostr.band"},
	MetadataRelays:   []string{"wss://nostr-pub.wellorder.net", "wss://purplepag.es", "wss://relay.nostr.band"},
}

func Init(ctx context.Context, keyOrBunker string, password string) error {
	sys.Init()

	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	if err := sys.InitSigner(ctx, keyOrBunker, &sdk.SignerOptions{Password: password}); err != nil {
		return err
	}

	return nil
}

type User struct {
	sdk.ProfileMetadata
	Groups []Group
}

func GetMe(ctx context.Context) User {
	pk := sys.Signer.GetPublicKey()

	me := User{
		ProfileMetadata: sys.FetchOrStoreProfileMetadata(ctx, pk),
		Groups:          make([]Group, 0, 100),
	}

	ie := sys.Pool.QuerySingle(ctx, sys.MetadataRelays, nostr.Filter{
		Kinds:   []int{10009},
		Authors: []string{me.PubKey},
	})
	if ie != nil {
		for _, tag := range ie.Tags {
			if len(tag) >= 2 && tag[0] == "group" {
				me.Groups = append(me.Groups, Group{tag[1], tag[2]})
			}
		}
	}

	return me
}

func GetUser(ctx context.Context, pubkey string) User {
	return User{
		ProfileMetadata: sys.FetchOrStoreProfileMetadata(ctx, pubkey),
		Groups:          nil,
	}
}
