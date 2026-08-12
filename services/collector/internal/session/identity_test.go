package session

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

// TestIdentityIsAPureValue is the assertion that makes Create's return
// safe to hand out at all.
//
// Create used to return the *State it had just stored, which `go test
// -race` reports as a data race against any concurrent With. The fix is
// to return a copy — but a copy is only a copy if every field is a
// value. State itself could not be copied this way: two of its three
// feature accumulators hold maps, so a shallow copy would share them
// and the fix would be cosmetic.
//
// So the guard is not "Create returns Identity" — the compiler says
// that. It is that Identity never acquires a field through which the
// store's memory could be reached.
func TestIdentityIsAPureValue(t *testing.T) {
	var reach func(reflect.Type, string)
	reach = func(rt reflect.Type, path string) {
		// time.Time is a leaf. It carries an internal *Location, which
		// this walk would otherwise flag — but that pointer addresses a
		// shared immutable zone, never anything the store owns, and the
		// standard library documents Time as safe to copy. Whitelisting
		// it by name rather than skipping all unexported fields, because
		// Identity lives in this package and an unexported map added
		// here later is exactly what this test is for.
		if rt.PkgPath() == "time" {
			return
		}
		switch rt.Kind() {
		case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Chan,
			reflect.Func, reflect.Interface, reflect.UnsafePointer:
			t.Errorf("Identity%s is a %s — copying it shares memory with "+
				"the store, which is the bug Identity exists to prevent",
				path, rt.Kind())
		case reflect.Struct:
			for i := 0; i < rt.NumField(); i++ {
				f := rt.Field(i)
				reach(f.Type, path+"."+f.Name)
			}
		case reflect.Array:
			reach(rt.Elem(), path+"[]")
		}
	}
	reach(reflect.TypeOf(Identity{}), "")
}

// TestCreateReturnsWhatTheStoreHolds guards the other half: a copy that
// is safe but wrong is no better than a pointer.
func TestCreateReturnsWhatTheStoreHolds(t *testing.T) {
	s := NewStore(time.Minute, time.Now)
	tok, ident, err := s.Create("t_acme", "/checkout", Client{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Built field by field from the live State, NOT via identity(). The
	// first version of this test compared identity() against itself,
	// which is true for any implementation — a mutation that zeroed
	// StartedAt passed it. A test whose two sides come from the same
	// function is not comparing anything.
	var held Identity
	if err := s.With(tok, func(st *State) {
		held = Identity{
			ID:        st.ID,
			TenantID:  st.TenantID,
			PagePath:  st.PagePath,
			StartedAt: st.StartedAt,
		}
	}); err != nil {
		t.Fatalf("With: %v", err)
	}
	if ident != held {
		t.Errorf("Create returned %+v, the store holds %+v", ident, held)
	}
	if ident.TenantID != "t_acme" || ident.PagePath != "/checkout" {
		t.Errorf("identity does not carry what Create was given: %+v", ident)
	}
	if ident.StartedAt.IsZero() || ident.ID == "" {
		t.Errorf("identity is missing a field Create must have set: %+v", ident)
	}
}

// TestConcurrentCreateAndWriteIsRaceFree exercises, under -race, the
// shape that used to be unsafe: a session being written while others are
// created. It passes trivially now — the point is that it fails loudly
// if Create is ever changed back to hand out live state.
func TestConcurrentCreateAndWriteIsRaceFree(t *testing.T) {
	s := NewStore(time.Minute, time.Now)
	tok, _, err := s.Create("t", "/", Client{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	const writers, batches = 8, 50
	var wg sync.WaitGroup
	wg.Add(writers + 1)
	for i := 0; i < writers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < batches; j++ {
				_ = s.With(tok, func(st *State) { st.ObserveBatch(uint32(j), uint32(j)) })
			}
		}()
	}
	go func() {
		defer wg.Done()
		for i := 0; i < batches; i++ {
			if _, _, err := s.Create("t", "/", Client{}); err != nil {
				t.Errorf("Create: %v", err)
				return
			}
		}
	}()
	wg.Wait()

	var seen uint32
	if err := s.With(tok, func(st *State) { seen = st.BatchesSeen }); err != nil {
		t.Fatalf("With: %v", err)
	}
	if want := uint32(writers * batches); seen != want {
		t.Errorf("BatchesSeen = %d, want %d — a lost update means the "+
			"store stopped serializing", seen, want)
	}
}
