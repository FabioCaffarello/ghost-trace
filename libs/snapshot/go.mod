module github.com/FabioCaffarello/ghost-trace/libs/snapshot

go 1.26

toolchain go1.26.5

require (
	github.com/FabioCaffarello/ghost-trace/libs/feature v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/genproto v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/policy v0.0.0
)

require google.golang.org/protobuf v1.36.11 // indirect

replace github.com/FabioCaffarello/ghost-trace/libs/feature => ../feature

replace github.com/FabioCaffarello/ghost-trace/libs/genproto => ../genproto

replace github.com/FabioCaffarello/ghost-trace/libs/policy => ../policy
