// The `jsonschema` struct tags are the published documentation of this
// API: cmd/gen-openapi reads them and nothing else describes the wire.
//
// They have one sharp edge. The library splits a tag on COMMAS, so a
// description containing one is silently truncated at it and the
// remainder is parsed as an option — an unknown option, which is then
// dropped without a word. The first version of these tags lost nine
// clause endings that way, including half of the paragraph explaining
// why observed_at is rejected rather than replaced.
//
// Nothing about that failure is visible: the build passes, the
// generator runs, and the specification simply says less than it was
// written to say. This test is the only thing standing between that
// and a published contract.
package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// parseNonTestFiles parses this package's non-test sources.
//
// parser.ParseDir would be shorter and is deprecated as of Go 1.25 —
// it ignores build tags when grouping files into packages. Reading the
// directory and parsing each file is the documented replacement that
// does not pull in golang.org/x/tools for a test helper.
func parseNonTestFiles(t *testing.T) []*ast.File {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package directory: %v", err)
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(".", name), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		files = append(files, f)
	}
	if len(files) == 0 {
		t.Fatal("parsed no source files — the guard is not guarding anything")
	}
	return files
}

// knownOptions is what invopop/jsonschema understands. A tag segment
// with any other key was produced by a stray comma.
var knownOptions = map[string]bool{
	"description":    true,
	"enum":           true,
	"format":         true,
	"minimum":        true,
	"maximum":        true,
	"minLength":      true,
	"maxLength":      true,
	"minItems":       true,
	"maxItems":       true,
	"pattern":        true,
	"default":        true,
	"example":        true,
	"title":          true,
	"required":       true,
	"readOnly":       true,
	"writeOnly":      true,
	"deprecated":     true,
	"nullable":       true,
	"oneof_required": true,
	"oneof_ref":      true,
	"oneof_type":     true,
}

func TestSchemaTagsHaveNoSwallowedText(t *testing.T) {
	for _, tag := range jsonschemaTagsInPackage(t) {
		for _, segment := range strings.Split(tag.value, ",") {
			key, _, hasValue := strings.Cut(segment, "=")
			if !hasValue {
				t.Errorf("%s: tag segment %q has no `=`, so it is being dropped.\n"+
					"  A comma inside a description truncates it — rewrite the prose without commas.\n"+
					"  full tag: %s", tag.field, segment, tag.value)
				continue
			}
			if !knownOptions[key] {
				t.Errorf("%s: tag segment %q uses unknown option %q, so it is being dropped.\n"+
					"  Almost always a comma inside a description — rewrite the prose without commas.\n"+
					"  full tag: %s", tag.field, segment, key, tag.value)
			}
		}
	}
}

// Every field that reaches the wire should say what it is. A missing
// description is a field a client author has to guess at.
func TestEveryWireFieldIsDescribed(t *testing.T) {
	for _, typ := range wireTypesUnderTest() {
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			if f.Tag.Get("json") == "" || f.Tag.Get("json") == "-" {
				continue
			}
			// A field whose type is another described struct carries its
			// documentation there.
			if f.Type.Kind() == reflect.Struct {
				continue
			}
			if !strings.Contains(f.Tag.Get("jsonschema"), "description=") {
				t.Errorf("%s.%s has no jsonschema description — it will appear in the "+
					"published contract as an undocumented field", typ.Name(), f.Name)
			}
		}
	}
}

func wireTypesUnderTest() []reflect.Type {
	return []reflect.Type{
		reflect.TypeOf(SessionsRequest{}),
		reflect.TypeOf(SessionsResponse{}),
		reflect.TypeOf(PageRef{}),
		reflect.TypeOf(ClientHints{}),
		reflect.TypeOf(CollectPolicy{}),
		reflect.TypeOf(TelemetryBatch{}),
		reflect.TypeOf(TelemetryPage{}),
		reflect.TypeOf(TelemetryEvent{}),
		reflect.TypeOf(DecisionsRequest{}),
		reflect.TypeOf(DecisionsResponse{}),
		reflect.TypeOf(DecisionReason{}),
		reflect.TypeOf(DecisionEvidence{}),
		reflect.TypeOf(OutcomesRequest{}),
		reflect.TypeOf(ErrorResponse{}),
	}
}

type taggedField struct {
	field string
	value string
}

// jsonschemaTagsInPackage reads the tags from SOURCE rather than by
// reflection, because reflect.StructTag.Get already parses away the
// malformation this test exists to find.
func jsonschemaTagsInPackage(t *testing.T) []taggedField {
	t.Helper()

	var out []taggedField
	for _, file := range parseNonTestFiles(t) {
		ast.Inspect(file, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				return true
			}
			for _, f := range st.Fields.List {
				if f.Tag == nil {
					continue
				}
				raw, err := strconv.Unquote(f.Tag.Value)
				if err != nil {
					continue
				}
				value, ok := reflect.StructTag(raw).Lookup("jsonschema")
				if !ok {
					continue
				}
				name := "?"
				if len(f.Names) > 0 {
					name = f.Names[0].Name
				}
				out = append(out, taggedField{
					field: ts.Name.Name + "." + name,
					value: value,
				})
			}
			return true
		})
	}
	if len(out) == 0 {
		t.Fatal("found no jsonschema tags — the guard is not guarding anything")
	}
	return out
}
