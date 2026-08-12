module github.com/FabioCaffarello/ghost-trace/services/collector

go 1.26

toolchain go1.26.5

require (
	google.golang.org/protobuf v1.36.11 // archive-format pin: canonical bytes are hashed for identity — upgrading is an archive-compatibility event, see libs/canonical
	lukechampine.com/blake3 v1.4.1 // indirect
	modernc.org/sqlite v1.56.0 // indirect
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
	github.com/klauspost/cpuid/v2 v2.0.9 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/sys v0.47.0 // indirect
	modernc.org/libc v1.74.4 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)

require (
	github.com/FabioCaffarello/ghost-trace/libs/decision v0.0.0-00010101000000-000000000000
	github.com/FabioCaffarello/ghost-trace/libs/id v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/metrics v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/tenant v0.0.0
	github.com/invopop/jsonschema v0.14.0
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3
	sigs.k8s.io/yaml v1.6.0
)

require (
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/buger/jsonparser v1.1.2 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/klauspost/compress v1.19.1 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nats-io/nats.go v1.52.0 // indirect
	github.com/nats-io/nkeys v0.4.15 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pb33f/ordered-map/v2 v2.3.1 // indirect
	github.com/prometheus/client_golang v1.24.1 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.70.1 // indirect
	github.com/prometheus/procfs v0.21.1 // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	go.yaml.in/yaml/v4 v4.0.0-rc.2 // indirect
	golang.org/x/crypto v0.54.0 // indirect
	golang.org/x/text v0.40.0 // indirect
)

// Filesystem replaces, kept alongside go.work on purpose: the
// workspace serves local development and CI, while these keep each
// module buildable ON ITS OWN — which is what the container build does
// (it copies libs/ and services/collector/ and never sees go.work).
replace github.com/FabioCaffarello/ghost-trace/libs/archive => ../../libs/archive

replace github.com/FabioCaffarello/ghost-trace/libs/canonical => ../../libs/canonical

replace github.com/FabioCaffarello/ghost-trace/libs/decision => ../../libs/decision

replace github.com/FabioCaffarello/ghost-trace/libs/id => ../../libs/id

replace github.com/FabioCaffarello/ghost-trace/libs/eventstream => ../../libs/eventstream

replace github.com/FabioCaffarello/ghost-trace/libs/feature => ../../libs/feature

replace github.com/FabioCaffarello/ghost-trace/libs/policy => ../../libs/policy

replace github.com/FabioCaffarello/ghost-trace/libs/snapshot => ../../libs/snapshot

replace github.com/FabioCaffarello/ghost-trace/libs/tenant => ../../libs/tenant

replace github.com/FabioCaffarello/ghost-trace/libs/genproto => ../../libs/genproto

replace github.com/FabioCaffarello/ghost-trace/libs/substrate => ../../libs/substrate

replace github.com/FabioCaffarello/ghost-trace/libs/wire => ../../libs/wire

replace github.com/FabioCaffarello/ghost-trace/libs/metrics => ../../libs/metrics

replace github.com/FabioCaffarello/ghost-trace/libs/middleware => ../../libs/middleware
