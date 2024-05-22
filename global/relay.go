package global

import (
	"context"

	neturl "net/url"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip11"
)

type Relay struct {
	URL   string
	Image string
	Name  string
}

func loadRelay(ctx context.Context, url string) *Relay {
	relay := &Relay{
		URL: nostr.NormalizeURL(url),
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
