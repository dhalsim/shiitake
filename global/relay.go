package global

import (
	"context"
	"sync"

	neturl "net/url"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip11"
	"github.com/puzpuzpuz/xsync/v3"
)

var relays = xsync.NewMapOf[string, *Relay]()

type Relay struct {
	URL   string
	Image string
	Name  string

	GroupsList   []*Group
	GroupsLoaded chan struct{}
}

var getRelayMutex sync.Mutex

func LoadRelay(ctx context.Context, url string) *Relay {
	getRelayMutex.Lock()
	defer getRelayMutex.Unlock()

	url = nostr.NormalizeURL(url)

	if relay, ok := relays.Load(url); ok {
		return relay
	}

	relay := &Relay{
		URL:          url,
		GroupsLoaded: make(chan struct{}),
	}

	info, _ := nip11.Fetch(ctx, url)
	if info != nil {
		relay.Image = info.Icon
		relay.Name = info.Name
	} else {
		parsed, _ := neturl.Parse(url)
		if parsed != nil {
			relay.Name = parsed.Host
		} else {
			relay.Name = url
		}
	}

	// TODO get all public groups in this relay

	return relay
}
