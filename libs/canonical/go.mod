module github.com/FabioCaffarello/ghost-trace/libs/canonical

go 1.26

toolchain go1.26.5

require (
	github.com/FabioCaffarello/ghost-trace/libs/genproto v0.0.0
	google.golang.org/protobuf v1.36.11 // archive-format pin: canonical bytes are hashed for identity — upgrading is an archive-compatibility event
	lukechampine.com/blake3 v1.4.1
)

require github.com/klauspost/cpuid/v2 v2.0.9 // indirect

replace github.com/FabioCaffarello/ghost-trace/libs/genproto => ../genproto
