module github.com/FabioCaffarello/ghost-trace/libs/decision

go 1.26

toolchain go1.26.5

require (
	github.com/FabioCaffarello/ghost-trace/libs/archive v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/feature v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/genproto v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/id v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/policy v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/snapshot v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/tenant v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/wire v0.0.0
	google.golang.org/protobuf v1.36.10
)

replace github.com/FabioCaffarello/ghost-trace/libs/archive => ../archive

replace github.com/FabioCaffarello/ghost-trace/libs/feature => ../feature

replace github.com/FabioCaffarello/ghost-trace/libs/genproto => ../genproto

replace github.com/FabioCaffarello/ghost-trace/libs/id => ../id

replace github.com/FabioCaffarello/ghost-trace/libs/policy => ../policy

replace github.com/FabioCaffarello/ghost-trace/libs/snapshot => ../snapshot

replace github.com/FabioCaffarello/ghost-trace/libs/tenant => ../tenant

replace github.com/FabioCaffarello/ghost-trace/libs/wire => ../wire
