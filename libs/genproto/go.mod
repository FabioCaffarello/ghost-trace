module github.com/FabioCaffarello/ghost-trace/libs/genproto

go 1.26

toolchain go1.26.5

require google.golang.org/protobuf v1.36.0 // archive-format pin: canonical bytes are hashed for identity — see services/ingestion/internal/canonical
