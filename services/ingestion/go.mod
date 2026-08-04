module github.com/FabioCaffarello/ghost-trace/services/ingestion

go 1.26

toolchain go1.26.5

require (
	google.golang.org/protobuf v1.36.10 // archive-format pin: canonical bytes are hashed for identity — upgrading is an archive-compatibility event, see internal/canonical
	lukechampine.com/blake3 v1.3.0 // indirect
	modernc.org/sqlite v1.34.4 // indirect
)

require (
	github.com/FabioCaffarello/ghost-trace/libs/archive v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/canonical v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/eventstream v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/feature v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/genproto v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/middleware v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/policy v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/snapshot v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/substrate v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/wire v0.0.0
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/klauspost/cpuid/v2 v2.0.9 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/sys v0.47.0 // indirect
	modernc.org/gc/v3 v3.0.0-20240107210532-573471604cb6 // indirect
	modernc.org/libc v1.55.3 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.8.0 // indirect
	modernc.org/strutil v1.2.0 // indirect
	modernc.org/token v1.1.0 // indirect
)

require (
	github.com/FabioCaffarello/ghost-trace/libs/decision v0.0.0-00010101000000-000000000000
	github.com/FabioCaffarello/ghost-trace/libs/id v0.0.0
	github.com/invopop/jsonschema v0.14.0
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2
	sigs.k8s.io/yaml v1.6.0
)

require (
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.2 // indirect
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/nats-io/nats.go v1.52.0 // indirect
	github.com/nats-io/nkeys v0.4.15 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pb33f/ordered-map/v2 v2.3.1 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	go.yaml.in/yaml/v4 v4.0.0-rc.2 // indirect
	golang.org/x/crypto v0.54.0 // indirect
	golang.org/x/text v0.40.0 // indirect
)

// Filesystem replaces, kept alongside go.work on purpose: the
// workspace serves local development and CI, while these keep each
// module buildable ON ITS OWN — which is what the container build does
// (it copies libs/ and services/ingestion/ and never sees go.work).
replace github.com/FabioCaffarello/ghost-trace/libs/archive => ../../libs/archive

replace github.com/FabioCaffarello/ghost-trace/libs/canonical => ../../libs/canonical

replace github.com/FabioCaffarello/ghost-trace/libs/decision => ../../libs/decision

replace github.com/FabioCaffarello/ghost-trace/libs/id => ../../libs/id

replace github.com/FabioCaffarello/ghost-trace/libs/eventstream => ../../libs/eventstream

replace github.com/FabioCaffarello/ghost-trace/libs/feature => ../../libs/feature

replace github.com/FabioCaffarello/ghost-trace/libs/policy => ../../libs/policy

replace github.com/FabioCaffarello/ghost-trace/libs/snapshot => ../../libs/snapshot

replace github.com/FabioCaffarello/ghost-trace/libs/genproto => ../../libs/genproto

replace github.com/FabioCaffarello/ghost-trace/libs/substrate => ../../libs/substrate

replace github.com/FabioCaffarello/ghost-trace/libs/wire => ../../libs/wire

replace github.com/FabioCaffarello/ghost-trace/libs/middleware => ../../libs/middleware
