package global

import (
	"context"
	"log"
	"sync"

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

	lastList            *nostr.Event
	listUpdateListeners []func()

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

	pubkey := sys.Signer.GetPublicKey()

	me = &Me{
		User: GetUser(ctx, pubkey),

		MetadataUpdated: make(chan struct{}),
		JoinedGroup:     make(chan *Group, 20),
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
			sys.StoreRelay.Publish(bg, *ie.Event)
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

		res, _ := sys.StoreRelay.QuerySync(bg, nostr.Filter{Kinds: []int{10009}, Authors: []string{me.PubKey}})
		if len(res) != 0 {
			processIncomingGroupListEvent(res[0])
			me.triggerListUpdate()
		}

		for ie := range sys.Pool.SubMany(bg, sys.FetchOutboxRelays(bg, me.PubKey), nostr.Filters{
			{
				Kinds:   []int{10009},
				Authors: []string{me.PubKey},
			},
		}) {
			processIncomingGroupListEvent(ie.Event)
			me.triggerListUpdate()
			sys.StoreRelay.Publish(bg, *ie.Event)
		}
	}()

	return me
}

func (me *Me) OnListUpdated(fn func()) {
	me.listUpdateListeners = append(me.listUpdateListeners, fn)
}

func (me *Me) triggerListUpdate() {
	for _, fn := range me.listUpdateListeners {
		fn()
	}
}
