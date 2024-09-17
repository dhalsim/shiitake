module fiatjaf.com/shiitake

go 1.23.0

require (
	fiatjaf.com/nostr-gtk v0.0.1
	github.com/bep/debounce v1.2.1
	github.com/diamondburned/adaptive v0.0.2-0.20221227093656-fa139be203a8
	github.com/diamondburned/arikawa/v3 v3.3.5
	github.com/diamondburned/chatkit v0.0.0-20240614105536-5788b19145bc
	github.com/diamondburned/gotk4-adwaita/pkg v0.0.0-20240604001651-1286f0db18ea
	github.com/diamondburned/gotk4/pkg v0.2.3-0.20240606221803-e395a91f5db3
	github.com/diamondburned/gotkit v0.0.0-20240614105032-cdfb37197d77
	github.com/dustin/go-humanize v1.0.1
	github.com/fiatjaf/eventstore v0.9.0
	github.com/ianlancetaylor/cgosymbolizer v0.0.0-20220405231054-a1ae3e4bba26
	github.com/mitchellh/go-homedir v1.1.0
	github.com/nbd-wtf/go-nostr v0.36.2
	github.com/puzpuzpuz/xsync/v3 v3.4.0
	golang.org/x/exp v0.0.0-20240909161429-701f63a606c0
	libdb.so/ctxt v0.0.0-20240229093153-2db38a5d3c12
)

require (
	github.com/KarpelesLab/weak v0.1.1 // indirect
	github.com/alecthomas/chroma v0.10.0 // indirect
	github.com/alessio/shellescape v1.4.1 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.3.4 // indirect
	github.com/btcsuite/btcd/btcutil v1.1.5 // indirect
	github.com/btcsuite/btcd/chaincfg/chainhash v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/danieljoos/wincred v1.1.0 // indirect
	github.com/decred/dcrd/crypto/blake256 v1.1.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.3.0 // indirect
	github.com/dgraph-io/badger/v4 v4.2.0 // indirect
	github.com/dgraph-io/ristretto v0.1.1 // indirect
	github.com/dlclark/regexp2 v1.4.0 // indirect
	github.com/fiatjaf/generic-ristretto v0.0.1 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.4.0 // indirect
	github.com/godbus/dbus/v5 v5.0.6 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/glog v1.2.1 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/flatbuffers v23.5.26+incompatible // indirect
	github.com/graph-gophers/dataloader/v7 v7.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/klauspost/compress v1.17.8 // indirect
	github.com/leonelquinteros/gotext v1.5.3-0.20230829162019-37f474cfb069 // indirect
	github.com/lmittmann/tint v1.0.4 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sahilm/fuzzy v0.1.1 // indirect
	github.com/tidwall/gjson v1.17.3 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/yalue/merged_fs v1.2.3 // indirect
	github.com/zalando/go-keyring v0.2.1 // indirect
	go.opencensus.io v0.24.0 // indirect
	go4.org/unsafe/assume-no-moving-gc v0.0.0-20231121144256-b99613f794b6 // indirect
	golang.org/x/crypto v0.27.0 // indirect
	golang.org/x/image v0.0.0-20220902085622-e7cb96979f69 // indirect
	golang.org/x/net v0.21.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
	golang.org/x/text v0.18.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
)

replace github.com/diamondburned/gotk4-adwaita/pkg => github.com/fiatjaf/gotk4-adwaita/pkg v0.0.0-20240604001651-1286f0db18ea

replace fiatjaf.com/nostr-gtk => ../nostr-gtk

replace github.com/nbd-wtf/go-nostr => ../go-nostr
