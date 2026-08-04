module github.com/FabioCaffarello/ghost-trace/services/archive

go 1.26

toolchain go1.26.5

require (
	github.com/FabioCaffarello/ghost-trace/libs/canonical v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/eventstream v0.0.0-00010101000000-000000000000
	github.com/FabioCaffarello/ghost-trace/libs/genproto v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/middleware v0.0.0-00010101000000-000000000000
	github.com/FabioCaffarello/ghost-trace/libs/substrate v0.0.0
	google.golang.org/protobuf v1.36.0
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/klauspost/cpuid/v2 v2.0.9 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/nats-io/nats.go v1.52.0 // indirect
	github.com/nats-io/nkeys v0.4.15 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/ncruces/go-strftime v0.1.9 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	golang.org/x/crypto v0.54.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	lukechampine.com/blake3 v1.3.0 // indirect
	modernc.org/gc/v3 v3.0.0-20240107210532-573471604cb6 // indirect
	modernc.org/libc v1.55.3 // indirect
	modernc.org/mathutil v1.6.0 // indirect
	modernc.org/memory v1.8.0 // indirect
	modernc.org/sqlite v1.34.4 // indirect
	modernc.org/strutil v1.2.0 // indirect
	modernc.org/token v1.1.0 // indirect
)

replace github.com/FabioCaffarello/ghost-trace/libs/canonical => ../../libs/canonical

replace github.com/FabioCaffarello/ghost-trace/libs/eventstream => ../../libs/eventstream

replace github.com/FabioCaffarello/ghost-trace/libs/genproto => ../../libs/genproto

replace github.com/FabioCaffarello/ghost-trace/libs/middleware => ../../libs/middleware

replace github.com/FabioCaffarello/ghost-trace/libs/substrate => ../../libs/substrate
