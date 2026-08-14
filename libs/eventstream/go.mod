module github.com/FabioCaffarello/ghost-trace/libs/eventstream

go 1.26

toolchain go1.26.6

require (
	github.com/FabioCaffarello/ghost-trace/libs/canonical v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/genproto v0.0.0
)

require (
	github.com/nats-io/nats.go v1.52.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/klauspost/cpuid/v2 v2.0.9 // indirect
	github.com/nats-io/nkeys v0.4.15 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.54.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	lukechampine.com/blake3 v1.4.1 // indirect
)

replace github.com/FabioCaffarello/ghost-trace/libs/canonical => ../canonical

replace github.com/FabioCaffarello/ghost-trace/libs/genproto => ../genproto
