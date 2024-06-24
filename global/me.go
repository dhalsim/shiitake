package global

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"sync"
	"time"

	"github.com/bep/debounce"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip29"
	sdk "github.com/nbd-wtf/nostr-sdk"
	"golang.org/x/exp/slices"
)

var (
	me     *Me
	meLock sync.Mutex
)

type Me struct {
	User

	lastList   *nostr.Event
	listUpdate struct {
		listeners []func()
		debouncer func(func())
	}

	MetadataUpdated chan struct{}
	JoinedGroup     chan *Group
	LeftGroup       chan nip29.GroupAddress
}

func (me *Me) InGroup(gad nip29.GroupAddress) bool {
	if me.lastList == nil {
		return false
	}
	for _, tag := range me.lastList.Tags {
		if len(tag) >= 3 && tag[0] == "group" && tag[1] == gad.ID && tag[2] == gad.Relay {
			return true
		}
	}
	return false
}

type User struct {
	sdk.ProfileMetadata
}

func GetMe(ctx context.Context) *Me {
	<-initialized

	meLock.Lock()
	defer meLock.Unlock()

	if me != nil {
		return me
	}

	pubkey := System.Signer.GetPublicKey()

	me = &Me{
		User: GetUser(ctx, pubkey),

		MetadataUpdated: make(chan struct{}),
		JoinedGroup:     make(chan *Group, 20),
		LeftGroup:       make(chan nip29.GroupAddress),
	}

	bg := context.Background()

	me.listUpdate.debouncer = debounce.New(700 * time.Millisecond)

	go func() {
		for ie := range System.Pool.SubMany(bg, System.MetadataRelays, nostr.Filters{
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
			System.StoreRelay.Publish(bg, *ie.Event)
			me.MetadataUpdated <- struct{}{}
		}
	}()

	go func() {
		// these are just for continued comparison with new events
		currentGroups := make([]nip29.GroupAddress, 0, 20)

		processIncomingGroupListEvent := func(evt *nostr.Event) {
			if me.lastList != nil && me.lastList.CreatedAt > evt.CreatedAt {
				// this event is older than the last one we have, ignore
				return
			}

			me.lastList = evt
			// every time a new list arrives we have to decide what groups were added and what groups were removed
			// and also modify the list of current groups

			// first we prepare a list of potentially removed groups with all the current
			removedGroups := make([]nip29.GroupAddress, len(currentGroups))
			copy(removedGroups, currentGroups)

			// then we go through the tags to see which groups were added and which ones were kept
			for _, tag := range evt.Tags {
				if len(tag) >= 2 && tag[0] == "group" {
					gad := nip29.GroupAddress{ID: tag[1], Relay: tag[2]}

					existing := slices.ContainsFunc(
						currentGroups, func(curr nip29.GroupAddress) bool { return curr.Equals(gad) })

					if !existing {
						// this is a new group
						currentGroups = append(currentGroups, gad)
						group := GetGroup(ctx, gad)
						if group != nil {
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
					}
				}
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

		res, _ := System.StoreRelay.QuerySync(bg, nostr.Filter{Kinds: []int{10009}, Authors: []string{me.PubKey}})
		if len(res) != 0 {
			processIncomingGroupListEvent(res[0])
			me.triggerListUpdate()
		}

		for ie := range System.Pool.SubMany(bg, System.FetchOutboxRelays(bg, me.PubKey), nostr.Filters{
			{
				Kinds:   []int{10009},
				Authors: []string{me.PubKey},
			},
		}) {
			processIncomingGroupListEvent(ie.Event)
			me.triggerListUpdate()
			System.StoreRelay.Publish(bg, *ie.Event)
		}
	}()

	return me
}

func (me *Me) OnListUpdated(fn func()) {
	me.listUpdate.listeners = append(me.listUpdate.listeners, fn)
}

func (me *Me) triggerListUpdate() {
	me.listUpdate.debouncer(func() {
		for _, fn := range me.listUpdate.listeners {
			fn()
		}
	})
}

func (me *Me) updateAndPublishLastList(ctx context.Context) error {
	me.lastList.CreatedAt = nostr.Now()
	if err := System.Signer.SignEvent(me.lastList); err != nil {
		return fmt.Errorf("failed to sign event: %w", err)
	}

	for _, url := range System.FetchOutboxRelays(ctx, me.PubKey) {
		relay, err := System.Pool.EnsureRelay(url)
		if err != nil {
			slog.Warn("failed to connect to outbox relay in order to publish list", "relay", url, "err", err)
			continue
		}

		if err := relay.Publish(ctx, *me.lastList); err != nil {
			slog.Warn("failed to publish groups list", "relay", url, "err", err)
			continue
		}
	}

	if err := System.StoreRelay.Publish(ctx, *me.lastList); err != nil {
		return fmt.Errorf("failed to store new groups list locally: %w", err)
	}

	me.triggerListUpdate()
	return nil
}
