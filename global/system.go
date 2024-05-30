package global

import (
	"context"
	"log"
	"slices"
	"sync"
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

type Me struct {
	User

	lastList *nostr.Event

	MetadataUpdated chan struct{}
	JoinedRelay     chan *Relay
	JoinedGroup     chan *Group
	LeftRelay       chan string
	LeftGroup       chan nip29.GroupAddress
}

type User struct {
	sdk.ProfileMetadata
}

var (
	me     *Me
	meLock sync.Mutex
)

func GetMe(ctx context.Context) *Me {
	meLock.Lock()
	defer meLock.Unlock()

	if me != nil {
		return me
	}

	pubkey := sys.Signer.GetPublicKey()

	me = &Me{
		User: GetUser(ctx, pubkey),

		MetadataUpdated: make(chan struct{}),
		JoinedRelay:     make(chan *Relay, 20),
		JoinedGroup:     make(chan *Group, 20),
		LeftRelay:       make(chan string),
		LeftGroup:       make(chan nip29.GroupAddress),
	}

	bg := context.Background()

	go func() {
		for ie := range sys.Pool.SubMany(bg, sys.MetadataRelays, nostr.Filters{
			{
				Kinds:   []int{0},
				Authors: []string{me.PubKey},
			},
		}) {
			if me.Event != nil && ie.Event.CreatedAt < me.Event.CreatedAt {
				continue
			}
			meta, err := sdk.ParseMetadata(ie.Event)
			if err != nil {
				continue
			}
			me.User.ProfileMetadata = meta
			me.MetadataUpdated <- struct{}{}
		}
	}()

	go func() {
		// these are just for continued comparison with new events
		currentGroups := make([]nip29.GroupAddress, 0, 20)

		// the relay list is just derived from the groups list in a pseudo-hierarchy
		currentRelays := make([]string, 0, 20)

		for ie := range sys.Pool.SubMany(bg, sys.FetchOutboxRelays(bg, me.PubKey), nostr.Filters{
			{
				Kinds:   []int{10009},
				Authors: []string{me.PubKey},
			},
		}) {
			if me.lastList != nil && me.lastList.CreatedAt > ie.CreatedAt {
				// this event is older than the last one we have, ignore
				continue
			}

			me.lastList = ie.Event
			// every time a new list arrives we have to decide what groups were added and what groups were removed
			// and also modify the list of current groups

			// first we prepare a list of potentially removed groups with all the current
			removedGroups := make([]nip29.GroupAddress, len(currentGroups))
			copy(removedGroups, currentGroups)

			// same for relays
			removedRelays := make([]string, len(currentRelays))
			copy(removedRelays, currentRelays)

			// then we go through the tags to see which groups were added and which ones were kept
			for _, tag := range ie.Tags {
				if len(tag) >= 2 && tag[0] == "group" {
					gad := nip29.GroupAddress{ID: tag[1], Relay: tag[2]}

					existing := slices.ContainsFunc(
						currentGroups, func(curr nip29.GroupAddress) bool { return curr.Equals(gad) })

					if !existing {
						// this is a new group
						currentGroups = append(currentGroups, gad)
						group := GetGroup(ctx, gad)
						if group != nil {
							// is it also a new relay?
							if !slices.Contains(currentRelays, gad.Relay) {
								currentRelays = append(currentRelays, gad.Relay)
								relay := loadRelay(ctx, gad.Relay)
								log.Printf("new relay: %s", gad.Relay)
								me.JoinedRelay <- relay
							}

							log.Printf("new group: %s", group.Address)
							me.JoinedGroup <- group // notify UI that this group was added
						}
					} else {
						// this is a group that was kept, therefore not deleted
						// so remove it from the list of groups that will be removed
						pos := slices.IndexFunc(
							removedGroups, func(curr nip29.GroupAddress) bool { return curr.Equals(gad) })
						if pos != -1 {
							// swap-remove
							removedGroups[pos] = removedGroups[len(removedGroups)-1]
							removedGroups = removedGroups[0 : len(removedGroups)-1]
						}

						// same for relays
						pos = slices.Index(removedRelays, gad.Relay)
						if pos != -1 {
							// swap-remove
							removedRelays[pos] = removedRelays[len(removedRelays)-1]
							removedRelays = removedRelays[0 : len(removedRelays)-1]
						}
					}

				}
			}

			// if a relay wasn't kept that means it was removed
			for _, url := range removedRelays {
				log.Printf("left relay: %s", url)
				// swap-remove
				pos := slices.Index(currentRelays, url)
				if pos != -1 {
					// swap-remove
					currentRelays[pos] = currentRelays[len(currentRelays)-1]
					currentRelays = currentRelays[0 : len(currentRelays)-1]
				}
				me.LeftRelay <- url // notify UI
			}

			// if a group wasn't kept that means it was removed
			for _, gad := range removedGroups {
				log.Printf("left group: %s", gad)
				// swap-remove
				pos := slices.IndexFunc(
					currentGroups, func(curr nip29.GroupAddress) bool { return curr.Equals(gad) })
				if pos != -1 {
					// swap-remove
					currentGroups[pos] = currentGroups[len(currentGroups)-1]
					currentGroups = currentGroups[0 : len(currentGroups)-1]
				}
				me.LeftGroup <- gad // notify UI
			}
		}
	}()

	return me
}

func GetUser(ctx context.Context, pubkey string) User {
	return User{
		ProfileMetadata: sys.FetchOrStoreProfileMetadata(ctx, pubkey),
	}
}
