package global

import (
	"context"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip29"
	sdk "github.com/nbd-wtf/nostr-sdk"
)

var sys = sdk.System()

func Init(ctx context.Context, keyOrBunker string, password string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	if err := sys.InitSigner(ctx, keyOrBunker, &sdk.SignerOptions{Password: password}); err != nil {
		return err
	}

	return nil
}

type User struct {
	sdk.ProfileMetadata
	Groups []*Group
}

var me *User

func GetMe(ctx context.Context) *User {
	if me != nil {
		return me
	}

	pk := sys.Signer.GetPublicKey()

	me = &User{
		ProfileMetadata: sys.FetchOrStoreProfileMetadata(ctx, pk),
		Groups:          make([]*Group, 0, 100),
	}

	ie := sys.Pool.QuerySingle(ctx, sys.MetadataRelays, nostr.Filter{
		Kinds:   []int{10009},
		Authors: []string{me.PubKey},
	})
	if ie != nil {
		for _, tag := range ie.Tags {
			if len(tag) >= 2 && tag[0] == "group" {
				group := GetGroup(ctx, nip29.GroupAddress{ID: tag[1], Relay: tag[2]})
				me.Groups = append(me.Groups, group)
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
