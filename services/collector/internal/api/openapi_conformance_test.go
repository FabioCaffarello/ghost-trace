// Does the published specification describe the API that actually
// runs?
//
// Generating the spec from the Go types makes it impossible for the
// SHAPES to drift. It does not make the hand-written half true: the
// status codes, the security schemes and which schema is declared for
// which response are written by a person in cmd/gen-openapi, and a
// person can be wrong. Saying a 400 returns ErrorResponse does not make
// it so.
//
// This closes that gap from the other end. Every committed golden — the
// literal bytes the server wrote — is validated against the schema the
// specification declares for that exact path, method and status. A spec
// that promises a field the server never sends, or a server that sends
// one the spec forbids, fails here.
package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"sigs.k8s.io/yaml"
)

// specPath is relative to this package: internal/api -> repository root.
const specPath = "../../../../contract/openapi.yaml"

// goldenResponses maps each committed golden to the response it is.
// The schema is NOT named here — it is looked up in the specification,
// which is the point: this asserts what the spec SAYS, not what we
// remember it saying.
var goldenResponses = []struct {
	golden string
	path   string
	method string
	status string
}{
	{"sessions_200.json", "/v1/sessions", "post", "200"},
	{"decisions_200.json", "/v1/decisions", "post", "200"},
	{"error_400_malformed.json", "/v1/sessions", "post", "400"},
	{"error_401_site_key.json", "/v1/sessions", "post", "401"},
	{"error_401_secret_key.json", "/v1/decisions", "post", "401"},
	{"error_400_action_required.json", "/v1/decisions", "post", "400"},
	{"error_400_evaluation_id_required.json", "/v1/outcomes", "post", "400"},
	{"error_400_unknown_outcome.json", "/v1/outcomes", "post", "400"},
	{"error_400_observed_at.json", "/v1/outcomes", "post", "400"},
	{"error_503_no_archive.json", "/v1/outcomes", "post", "503"},
	{"error_400_telemetry_malformed.json", "/v1/telemetry", "post", "400"},
	{"error_401_outcomes_secret_key.json", "/v1/outcomes", "post", "401"},
}

// A 500 needs an induced internal failure and there is no seam to
// induce one through — the paths that produce it are a crypto/rand
// read and a storage write. They are exempted EXPLICITLY rather than
// quietly missing, and the exemption is narrow: every 500 body goes
// through the same writeError and the same ErrorResponse type that the
// seven error goldens above already freeze, so what is unproven here is
// the status code, not the shape.
var untestableResponses = map[string]string{
	"post /v1/sessions 500":  "requires crypto/rand to fail",
	"post /v1/telemetry 500": "requires the ingest path to fail internally",
	"post /v1/decisions 500": "requires policy evaluation or id minting to fail",
	"post /v1/outcomes 500":  "requires the archive write to fail for a non-availability reason",
}

func TestGoldensConformToPublishedSpec(t *testing.T) {
	doc := loadSpec(t)
	compiler := jsonschema.NewCompiler()
	// An absolute URI, not a bare filename: with a relative one the
	// compiler resolves the document's internal "#/components/..."
	// references against the working directory and tries to open them
	// as files. Only the schemas containing no $ref appeared to work.
	const specURI = "https://ghost-trace.invalid/openapi.json"
	if err := compiler.AddResource(specURI, doc); err != nil {
		t.Fatalf("add spec as schema resource: %v", err)
	}

	for _, tc := range goldenResponses {
		t.Run(tc.golden, func(t *testing.T) {
			ref := declaredSchemaRef(t, doc, tc.path, tc.method, tc.status)

			schema, err := compiler.Compile(specURI + "#" + jsonPointerOf(ref))
			if err != nil {
				t.Fatalf("compile %s: %v", ref, err)
			}

			raw, err := os.ReadFile(filepath.Join(goldenDir, tc.golden))
			if err != nil {
				t.Fatalf("read golden: %v", err)
			}
			var instance any
			if err := json.Unmarshal(raw, &instance); err != nil {
				t.Fatalf("golden is not JSON: %v", err)
			}

			if err := schema.Validate(instance); err != nil {
				t.Errorf("the bytes the server actually returns do not satisfy the schema\n"+
					"the specification declares for %s %s -> %s (%s):\n\n%v\n\nbytes: %s",
					tc.method, tc.path, tc.status, ref, err, raw)
			}
		})
	}
}

// Every response the spec declares with a JSON body should have a
// golden. Without this, adding an endpoint silently adds an undescribed
// response and the table above quietly stops being complete.
func TestEveryDeclaredJSONResponseHasAGolden(t *testing.T) {
	doc := loadSpec(t)
	covered := map[string]bool{}
	for _, tc := range goldenResponses {
		covered[tc.method+" "+tc.path+" "+tc.status] = true
	}

	paths, _ := doc["paths"].(map[string]any)
	for path, item := range paths {
		ops, _ := item.(map[string]any)
		for method, opAny := range ops {
			op, _ := opAny.(map[string]any)
			responses, _ := op["responses"].(map[string]any)
			for status, respAny := range responses {
				resp, _ := respAny.(map[string]any)
				content, _ := resp["content"].(map[string]any)
				if _, isJSON := content["application/json"]; !isJSON {
					continue // 202s carry no body; healthz is text
				}
				key := method + " " + path + " " + status
				if _, exempt := untestableResponses[key]; exempt {
					continue
				}
				if !covered[key] {
					t.Errorf("%s declares a JSON response with no golden proving the "+
						"server actually produces it — add it to goldenResponses", key)
				}
			}
		}
	}
}

func loadSpec(t *testing.T) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read %s (generate it with `make openapi`): %v", specPath, err)
	}
	asJSON, err := yaml.YAMLToJSON(raw)
	if err != nil {
		t.Fatalf("spec is not valid YAML: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(asJSON, &doc); err != nil {
		t.Fatalf("spec is not a JSON object: %v", err)
	}
	return doc
}

// declaredSchemaRef digs the $ref out of
// paths -> <path> -> <method> -> responses -> <status> ->
// content -> application/json -> schema.
func declaredSchemaRef(t *testing.T, doc map[string]any, path, method, status string) string {
	t.Helper()

	step := func(node any, key, what string) any {
		m, ok := node.(map[string]any)
		if !ok {
			t.Fatalf("%s %s -> %s: %s is not an object", method, path, status, what)
		}
		v, ok := m[key]
		if !ok {
			t.Fatalf("%s %s -> %s: the specification declares no %s", method, path, status, what)
		}
		return v
	}

	node := step(doc, "paths", "paths")
	node = step(node, path, "path "+path)
	node = step(node, method, "method "+method)
	node = step(node, "responses", "responses")
	node = step(node, status, "status "+status)
	node = step(node, "content", "content")
	node = step(node, "application/json", "application/json content")
	node = step(node, "schema", "schema")

	schema, _ := node.(map[string]any)
	ref, ok := schema["$ref"].(string)
	if !ok {
		t.Fatalf("%s %s -> %s: schema is inline, expected a $ref into components",
			method, path, status)
	}
	return ref
}

// jsonPointerOf turns "#/components/schemas/X" into "/components/schemas/X".
func jsonPointerOf(ref string) string {
	if len(ref) > 0 && ref[0] == '#' {
		return ref[1:]
	}
	return ref
}
