module github.com/FabioCaffarello/ghost-trace/libs/eventstream

go 1.26

toolchain go1.26.5

require github.com/FabioCaffarello/ghost-trace/libs/genproto v0.0.0

require google.golang.org/protobuf v1.36.0

replace github.com/FabioCaffarello/ghost-trace/libs/genproto => ../genproto
