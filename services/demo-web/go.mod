module github.com/FabioCaffarello/ghost-trace/services/demo-web

go 1.26

toolchain go1.26.5

require (
	github.com/FabioCaffarello/ghost-trace/libs/archive v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/decision v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/metrics v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/middleware v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/policy v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/tenant v0.0.0
	github.com/FabioCaffarello/ghost-trace/libs/wire v0.0.0
)

require (
	github.com/FabioCaffarello/ghost-trace/libs/feature v0.0.0 // indirect
	github.com/FabioCaffarello/ghost-trace/libs/genproto v0.0.0 // indirect
	github.com/FabioCaffarello/ghost-trace/libs/id v0.0.0 // indirect
	github.com/FabioCaffarello/ghost-trace/libs/snapshot v0.0.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_golang v1.24.1 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.70.1 // indirect
	github.com/prometheus/procfs v0.21.1 // indirect
	golang.org/x/sys v0.47.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/FabioCaffarello/ghost-trace/libs/archive => ../../libs/archive

replace github.com/FabioCaffarello/ghost-trace/libs/decision => ../../libs/decision

replace github.com/FabioCaffarello/ghost-trace/libs/feature => ../../libs/feature

replace github.com/FabioCaffarello/ghost-trace/libs/genproto => ../../libs/genproto

replace github.com/FabioCaffarello/ghost-trace/libs/id => ../../libs/id

replace github.com/FabioCaffarello/ghost-trace/libs/metrics => ../../libs/metrics

replace github.com/FabioCaffarello/ghost-trace/libs/middleware => ../../libs/middleware

replace github.com/FabioCaffarello/ghost-trace/libs/policy => ../../libs/policy

replace github.com/FabioCaffarello/ghost-trace/libs/snapshot => ../../libs/snapshot

replace github.com/FabioCaffarello/ghost-trace/libs/tenant => ../../libs/tenant

replace github.com/FabioCaffarello/ghost-trace/libs/wire => ../../libs/wire
