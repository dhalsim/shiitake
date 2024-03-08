package global

import (
	"context"

	"github.com/nbd-wtf/go-nostr"
	sdk "github.com/nbd-wtf/nostr-sdk"
	cache_memory "github.com/nbd-wtf/nostr-sdk/cache/memory"
)

var Sys = sdk.System{
	Pool:             nostr.NewSimplePool(context.Background()),
	RelaysCache:      cache_memory.New32[[]sdk.Relay](1000),
	MetadataCache:    cache_memory.New32[sdk.ProfileMetadata](1000),
	FollowsCache:     cache_memory.New32[[]sdk.Follow](1),
	RelayListRelays:  []string{"wss://purplepag.es", "wss://relay.nostr.band"},
	FollowListRelays: []string{"wss://public.relaying.io", "wss://relay.nostr.band"},
	MetadataRelays:   []string{"wss://nostr-pub.wellorder.net", "wss://purplepag.es", "wss://relay.nostr.band"},
}

func GetMe(ctx context.Context) sdk.ProfileMetadata {
	pk := Sys.Signer.GetPublicKey()
	return Sys.FetchOrStoreProfileMetadata(ctx, pk)
}
