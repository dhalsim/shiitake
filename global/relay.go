package global

import (
	"context"
	"fmt"
	"log/slog"
	neturl "net/url"
	"sync"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip11"
	"github.com/nbd-wtf/go-nostr/nip29"
	"github.com/puzpuzpuz/xsync/v3"
)

var relays = xsync.NewMapOf[string, *Relay]()

type Relay struct {
	URL   string
	Image string
	Name  string

	GroupsList   []nip29.Group
	GroupsLoaded chan struct{}
}

var getRelayMutex sync.Mutex

func LoadRelay(ctx context.Context, url string) (*Relay, error) {
	getRelayMutex.Lock()
	defer getRelayMutex.Unlock()

	url = nostr.NormalizeURL(url)
	if relay, ok := relays.Load(url); ok {
		return relay, nil
	}

	relay := &Relay{
		URL:          url,
		GroupsLoaded: make(chan struct{}),
	}

	info, err := nip11.Fetch(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to get information from '%s': %w", url, err)
	}

	relay.Image = info.Icon
	relay.Name = info.Name
	parsed, _ := neturl.Parse(url)
	if parsed != nil {
		relay.Name = parsed.Host
	} else {
		relay.Name = url
	}

	r, err := sys.Pool.EnsureRelay(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to '%s': %w", url, err)
	}

	// get all public groups in this relay
	go func() {
		res, _ := r.QuerySync(ctx, nostr.Filter{Kinds: []int{39000}, Limit: 10})
		relay.GroupsList = make([]nip29.Group, 0, len(res))
		for _, evt := range res {
			group, err := nip29.NewGroupFromMetadataEvent(url, evt)
			if err != nil {
				slog.Warn("invalid group metadata received", "event", evt)
				continue
			}
			relay.GroupsList = append(relay.GroupsList, group)
		}
		close(relay.GroupsLoaded)
	}()

	return relay, nil
}
