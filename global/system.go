package global

import (
	"context"
	"time"

	"github.com/fiatjaf/eventstore/badger"
	"github.com/mitchellh/go-homedir"
	sdk "github.com/nbd-wtf/nostr-sdk"
)

var (
	path, _ = homedir.Expand("~/.local/share/shiitake/cachedb")
	bb      = &badger.BadgerBackend{Path: path}
	_       = bb.Init()
	System  = sdk.NewSystem(sdk.WithStore(bb))

	initialized = make(chan struct{})
)

func Init(ctx context.Context, keyOrBunker string, password string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	if err := System.InitSigner(ctx, keyOrBunker, &sdk.SignerOptions{Password: password}); err != nil {
		return err
	}

	close(initialized)

	return nil
}

func GetUser(ctx context.Context, pubkey string) User {
	return User{
		ProfileMetadata: System.FetchOrStoreProfileMetadata(ctx, pubkey),
	}
}
