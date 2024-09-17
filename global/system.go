package global

import (
	"context"
	"time"

	"github.com/fiatjaf/eventstore/badger"
	"github.com/mitchellh/go-homedir"
	"github.com/nbd-wtf/go-nostr/keyer"
	"github.com/nbd-wtf/go-nostr/sdk"
)

var (
	path, _ = homedir.Expand("~/.local/share/shiitake/cachedb")
	bb      = &badger.BadgerBackend{Path: path}
	_       = bb.Init()
	System  = sdk.NewSystem(sdk.WithStore(bb))
	K       keyer.Keyer

	initialized = make(chan struct{})
)

func Init(ctx context.Context, keyOrBunker string, password string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	if k, err := keyer.New(ctx, System.Pool, keyOrBunker, &keyer.SignerOptions{Password: password}); err != nil {
		return err
	} else {
		K = k
	}

	close(initialized)

	return nil
}

func GetUser(ctx context.Context, pubkey string) User {
	return User{
		ProfileMetadata: System.FetchProfileMetadata(ctx, pubkey),
	}
}
