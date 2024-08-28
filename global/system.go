package global

import (
	"context"
	"time"

	"github.com/fiatjaf/eventstore/badger"
	"github.com/mitchellh/go-homedir"
	sdk "github.com/nbd-wtf/nostr-sdk"
	"github.com/nbd-wtf/nostr-sdk/signer"
)

var (
	path, _ = homedir.Expand("~/.local/share/shiitake/cachedb")
	bb      = &badger.BadgerBackend{Path: path}
	_       = bb.Init()
	System  = sdk.NewSystem(sdk.WithStore(bb))
	Signer  signer.Signer

	initialized = make(chan struct{})
)

func Init(ctx context.Context, keyOrBunker string, password string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	if s, err := signer.New(ctx, System.Pool, keyOrBunker, &signer.SignerOptions{Password: password}); err != nil {
		return err
	} else {
		Signer = s
	}

	close(initialized)

	return nil
}

func GetUser(ctx context.Context, pubkey string) User {
	return User{
		ProfileMetadata: System.FetchProfileMetadata(ctx, pubkey),
	}
}
