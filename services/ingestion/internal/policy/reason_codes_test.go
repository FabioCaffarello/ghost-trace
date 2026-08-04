// The §7 stable-enumeration invariant, made executable.
//
// ReasonCodes is what the published contract advertises. If a new
// Reason* constant is added and not listed there, the OpenAPI schema
// silently under-describes the API and a client written from the spec
// will meet a code it was told could not occur. If a code is listed
// but no longer produced, the spec advertises something that will
// never arrive.
//
// Both directions are asserted by PARSING THIS PACKAGE'S SOURCE rather
// than by a second hand-written list, because a hand-written list is
// the thing being guarded against.
package policy

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
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

// reasonConstsFromSource returns every `Reason<Name> = "CODE"` constant
// declared in this package's non-test files, as name -> value.
func reasonConstsFromSource(t *testing.T) map[string]string {
	t.Helper()

	found := map[string]string{}
	for _, file := range parseNonTestFiles(t) {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok || len(vs.Names) != 1 || len(vs.Values) != 1 {
					continue
				}
				name := vs.Names[0].Name
				if !strings.HasPrefix(name, "Reason") {
					continue
				}
				lit, ok := vs.Values[0].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				val, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatalf("unquote %s: %v", name, err)
				}
				found[name] = val
			}
		}
	}
	return found
}

func TestReasonCodesCoverEveryConstant(t *testing.T) {
	consts := reasonConstsFromSource(t)
	if len(consts) == 0 {
		t.Fatal("parsed no Reason* constants — the guard is not guarding anything")
	}

	published := map[string]bool{}
	for _, c := range ReasonCodes {
		published[c] = true
	}

	for name, value := range consts {
		if !published[value] {
			t.Errorf("%s = %q is not in ReasonCodes — the published contract "+
				"does not mention a code the service can return", name, value)
		}
	}

	declared := map[string]bool{}
	for _, value := range consts {
		declared[value] = true
	}
	for _, c := range ReasonCodes {
		if !declared[c] {
			t.Errorf("ReasonCodes lists %q, which no Reason* constant declares — "+
				"the contract advertises a code that can never arrive", c)
		}
	}

	if len(ReasonCodes) != len(consts) {
		t.Errorf("ReasonCodes has %d entries, source declares %d constants",
			len(ReasonCodes), len(consts))
	}
}

// A duplicate would make the OpenAPI enum wrong in a way the set
// comparisons above cannot see.
func TestReasonCodesHasNoDuplicates(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range ReasonCodes {
		if seen[c] {
			t.Errorf("ReasonCodes lists %q more than once", c)
		}
		seen[c] = true
	}
}
