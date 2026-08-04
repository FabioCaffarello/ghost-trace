// Package sdk serves the browser SDK.
//
// It lives with the COLLECTOR, not with the demo host, because the SDK
// is Ghost Trace's artefact and not the customer's. A real integrator
// puts <script src="https://collector.example/sdk.js"> on their page;
// they do not vendor a copy, and if they did, every deployment would be
// running a different version of the thing that decides what the wire
// carries.
//
// That is also why internal/ingest/vocabulary_test.go can read this
// file: the published vocabularies and the code that emits them are one
// module apart from nothing at all. When the SDK briefly moved out with
// the demo page, that guard failed loudly rather than quietly passing —
// which is the only reason this is where it is.
//
// A classic script tag is not subject to CORS, so serving it to another
// origin needs nothing beyond this.
package sdk

import (
	_ "embed"
	"net/http"
)

//go:embed sdk.js
var script []byte

// Register mounts GET /sdk.js on mux.
func Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /sdk.js", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(script)
	})
}
