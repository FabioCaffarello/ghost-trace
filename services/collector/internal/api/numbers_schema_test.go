// Cross-check: the hand-written validator in experiments/schema agrees
// with a real JSON Schema implementation.
//
// numbers.py refuses to publish a numbers.json that does not satisfy
// experiments/schema/numbers.schema.json, and it checks that with a
// stdlib validator — because numbers.py is stdlib-only by design and
// reproducing the six numbers must not require a dependency install.
//
// A hand-written validator that quietly accepts what it does not
// understand is worse than none, since it reports success. This test is
// the other half of the answer: it runs the SAME committed fixture
// corpus through santhosh-tekuri/jsonschema and asserts both reach the
// same verdict on every fixture. If the Python validator is wrong about
// a case that matters, this is what says so.
//
// It lives here rather than in experiments/ because this module already
// has the validator, from R1.14's OpenAPI conformance test.
package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Relative to this package: internal/api -> repository root.
const (
	numbersSchemaPath = "../../../../experiments/schema/numbers.schema.json"
	numbersFixtureDir = "../../../../experiments/schema/testdata"
)

func TestNumbersSchemaFixturesAgreeWithRealValidator(t *testing.T) {
	compiler := jsonschema.NewCompiler()

	raw, err := os.ReadFile(numbersSchemaPath)
	if err != nil {
		t.Fatalf("read numbers schema: %v", err)
	}
	var schemaDoc any
	if err := json.Unmarshal(raw, &schemaDoc); err != nil {
		t.Fatalf("numbers schema is not JSON: %v", err)
	}
	const uri = "https://ghost-trace.invalid/numbers.schema.json"
	if err := compiler.AddResource(uri, schemaDoc); err != nil {
		t.Fatalf("add numbers schema: %v", err)
	}
	schema, err := compiler.Compile(uri)
	if err != nil {
		t.Fatalf("compile numbers schema: %v", err)
	}

	var valid, invalid int
	for _, expectation := range []string{"valid", "invalid"} {
		wantValid := expectation == "valid"
		dir := filepath.Join(numbersFixtureDir, expectation)

		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			name := e.Name()
			t.Run(expectation+"/"+name, func(t *testing.T) {
				body, err := os.ReadFile(filepath.Join(dir, name))
				if err != nil {
					t.Fatalf("read fixture: %v", err)
				}
				var doc any
				if err := json.Unmarshal(body, &doc); err != nil {
					t.Fatalf("fixture is not JSON: %v", err)
				}

				err = schema.Validate(doc)
				gotValid := err == nil

				switch {
				case wantValid && !gotValid:
					t.Errorf("experiments/schema/testdata/valid/%s is rejected by a real "+
						"JSON Schema validator, so the Python validator is too permissive "+
						"or the schema is wrong:\n%v", name, err)
				case !wantValid && gotValid:
					t.Errorf("experiments/schema/testdata/invalid/%s is ACCEPTED by a real "+
						"JSON Schema validator — the Python validator rejects it, so the "+
						"two disagree about what the contract means", name)
				}
			})
			if wantValid {
				valid++
			} else {
				invalid++
			}
		}
	}

	// A corpus that quietly emptied would make this test pass while
	// checking nothing at all.
	if valid == 0 || invalid == 0 {
		t.Fatalf("fixture corpus is one-sided: %d valid, %d invalid", valid, invalid)
	}
}
