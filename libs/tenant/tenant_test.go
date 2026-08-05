package tenant_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/libs/tenant"
)

func two(t *testing.T) *tenant.Registry {
	t.Helper()
	r, err := tenant.New(
		tenant.Tenant{ID: "t_a", SiteKey: "pk_a", SecretKey: "sk_a"},
		tenant.Tenant{ID: "t_b", SiteKey: "pk_b", SecretKey: "sk_b"},
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return r
}

func TestKeysResolveToTheirOwnTenant(t *testing.T) {
	r := two(t)
	for _, tc := range []struct{ site, secret, want string }{
		{"pk_a", "sk_a", "t_a"},
		{"pk_b", "sk_b", "t_b"},
	} {
		if got, ok := r.BySiteKey(tc.site); !ok || got.ID != tc.want {
			t.Errorf("BySiteKey(%q) = %v, want %s", tc.site, got, tc.want)
		}
		if got, ok := r.BySecretKey(tc.secret); !ok || got.ID != tc.want {
			t.Errorf("BySecretKey(%q) = %v, want %s", tc.secret, got, tc.want)
		}
	}
}

func TestOneTenantsSecretDoesNotResolveAnother(t *testing.T) {
	// The whole point. If this ever held, a caller with one valid
	// credential could act for every customer in the file.
	r := two(t)
	got, ok := r.BySecretKey("sk_a")
	if !ok || got.ID != "t_a" {
		t.Fatalf("BySecretKey(sk_a) = %v", got)
	}
	if got.SiteKey != "pk_a" {
		t.Errorf("sk_a resolved to a tenant carrying %q", got.SiteKey)
	}
}

func TestAnUnknownKeyResolvesToNothing(t *testing.T) {
	r := two(t)
	if _, ok := r.BySecretKey("sk_nope"); ok {
		t.Error("an unknown secret resolved to a tenant")
	}
	if _, ok := r.BySiteKey("pk_nope"); ok {
		t.Error("an unknown site key resolved to a tenant")
	}
	// An empty bearer is the commonest unauthenticated request there
	// is; it must not match an entry, and it must not match by accident
	// if someone later writes an entry with an empty secret.
	if _, ok := r.BySecretKey(""); ok {
		t.Error("an empty secret resolved to a tenant")
	}
}

func TestASharedSecretIsRefusedAtStartup(t *testing.T) {
	// Two tenants with one secret means either can act as the other,
	// and both requests authenticate — the failure is invisible at
	// request time, so it has to be caught here.
	_, err := tenant.New(
		tenant.Tenant{ID: "t_a", SiteKey: "pk_a", SecretKey: "same"},
		tenant.Tenant{ID: "t_b", SiteKey: "pk_b", SecretKey: "same"},
	)
	if err == nil {
		t.Fatal("a shared secret_key was accepted")
	}
	if !strings.Contains(err.Error(), "shared") {
		t.Errorf("err = %v, want it to name the sharing", err)
	}
}

func TestOtherCollisionsAreRefused(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   []tenant.Tenant
	}{
		{"duplicate id", []tenant.Tenant{
			{ID: "t", SiteKey: "pk_a", SecretKey: "sk_a"},
			{ID: "t", SiteKey: "pk_b", SecretKey: "sk_b"}}},
		{"duplicate site key", []tenant.Tenant{
			{ID: "t_a", SiteKey: "pk", SecretKey: "sk_a"},
			{ID: "t_b", SiteKey: "pk", SecretKey: "sk_b"}}},
		{"no id", []tenant.Tenant{{SiteKey: "pk", SecretKey: "sk"}}},
		{"no site key", []tenant.Tenant{{ID: "t", SecretKey: "sk"}}},
		{"no secret key", []tenant.Tenant{{ID: "t", SiteKey: "pk"}}},
	} {
		if _, err := tenant.New(tc.in...); err == nil {
			t.Errorf("%s was accepted", tc.name)
		}
	}
}

func TestAnEmptyRegistryIsRefused(t *testing.T) {
	// A service that speaks for nobody would accept nobody, and would
	// do it by answering 401 to every request — which looks like a key
	// problem rather than a configuration one.
	if _, err := tenant.New(); err == nil {
		t.Error("an empty registry was accepted")
	}
}

func write(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "tenants.json")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return p
}

func TestLoadReadsAFile(t *testing.T) {
	r, err := tenant.Load(write(t, `{"tenants":[
		{"id":"t_a","site_key":"pk_a","secret_key":"sk_a"},
		{"id":"t_b","site_key":"pk_b","secret_key":"sk_b"}]}`))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if r.Len() != 2 {
		t.Errorf("Len = %d, want 2", r.Len())
	}
}

func TestLoadRejectsAnUnknownField(t *testing.T) {
	// A typo in a security-relevant file. "secret" instead of
	// "secret_key" would otherwise leave the secret empty and be
	// rejected for the wrong reason; a typo'd id would not be rejected
	// at all.
	_, err := tenant.Load(write(t, `{"tenants":[
		{"id":"t_a","site_key":"pk_a","secret_key":"sk_a","sekret":"x"}]}`))
	if err == nil {
		t.Error("an unknown field was accepted")
	}
}

func TestIDsNeverCarrySecrets(t *testing.T) {
	// IDs() exists so a process can log what it serves. If it ever
	// carried the credential, every deployment would print its own
	// secrets at startup.
	for _, id := range two(t).IDs() {
		if strings.Contains(id, "sk_") {
			t.Errorf("IDs() returned %q, which looks like a secret", id)
		}
	}
}

func TestFingerprintIsStableAndOrderIndependent(t *testing.T) {
	// It is compared across processes, so declaration order in a file
	// must not change it — otherwise two identical registries would
	// read as a disagreement and the check would be muted for crying
	// wolf.
	a, err := tenant.New(
		tenant.Tenant{ID: "t_a", SiteKey: "pk_a", SecretKey: "sk_a"},
		tenant.Tenant{ID: "t_b", SiteKey: "pk_b", SecretKey: "sk_b"},
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	b, err := tenant.New(
		tenant.Tenant{ID: "t_b", SiteKey: "pk_b", SecretKey: "sk_b"},
		tenant.Tenant{ID: "t_a", SiteKey: "pk_a", SecretKey: "sk_a"},
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if a.Fingerprint() != b.Fingerprint() {
		t.Errorf("the same tenants in a different order fingerprint differently: %s vs %s",
			a.Fingerprint(), b.Fingerprint())
	}
}

func TestFingerprintChangesWhenTheTenantSetDoes(t *testing.T) {
	base := two(t)
	extra, err := tenant.New(
		tenant.Tenant{ID: "t_a", SiteKey: "pk_a", SecretKey: "sk_a"},
		tenant.Tenant{ID: "t_b", SiteKey: "pk_b", SecretKey: "sk_b"},
		tenant.Tenant{ID: "t_c", SiteKey: "pk_c", SecretKey: "sk_c"},
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if base.Fingerprint() == extra.Fingerprint() {
		t.Error("adding a tenant did not change the fingerprint; the check would " +
			"pass while two services disagreed about who exists")
	}
}

func TestFingerprintCarriesNoSecret(t *testing.T) {
	// It is published on an unauthenticated endpoint. Two registries
	// differing only in their secrets must fingerprint the SAME, which
	// is the proof that no secret went into it.
	a, err := tenant.New(tenant.Tenant{ID: "t", SiteKey: "pk", SecretKey: "sk_one"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	b, err := tenant.New(tenant.Tenant{ID: "t", SiteKey: "pk", SecretKey: "sk_two"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if a.Fingerprint() != b.Fingerprint() {
		t.Error("the fingerprint depends on the secret; it is published unauthenticated " +
			"and its safety would then rest on secret entropy this package cannot know")
	}
	if strings.Contains(a.Fingerprint(), "sk_") {
		t.Error("the fingerprint contains a secret verbatim")
	}
}
