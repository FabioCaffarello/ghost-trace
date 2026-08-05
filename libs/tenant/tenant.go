// Package tenant is the registry that turns a key into a tenant.
//
// Until now a tenant was PROCESS CONFIGURATION: one -tenant flag, one
// -site-key, one -secret-key, and every session the process saw
// belonged to the same customer by construction. That is a deployment
// per customer, not multi-tenancy, and it hid a question the split made
// urgent — with sessions in a shared KV bucket and records on a shared
// stream, WHO a request speaks for has to be derived from what the
// request carries.
//
// site_key identifies and does not authenticate (contract §1): it is
// public, in the page source, and a wrong one only stops cross-tenant
// noise. secret_key authenticates the application server, so it is what
// the decision endpoints resolve a tenant FROM — a caller cannot claim
// a tenant it has no secret for, because claiming and proving are the
// same act.
package tenant

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Tenant is one customer.
type Tenant struct {
	// ID is what every archived record is attributed to and what keys
	// the snapshot bucket.
	ID string `json:"id"`

	// SiteKey is public and embedded in the page.
	SiteKey string `json:"site_key"`

	// SecretKey authenticates this tenant's application server. It is
	// never rendered, logged, or returned.
	SecretKey string `json:"secret_key"`
}

// Registry resolves keys to tenants.
type Registry struct {
	all    []Tenant
	bySite map[string]*Tenant
}

type file struct {
	Tenants []Tenant `json:"tenants"`
}

// Load reads a registry from JSON.
func Load(path string) (*Registry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("tenant: read %s: %w", path, err)
	}
	var f file
	dec := json.NewDecoder(bytes.NewReader(raw))
	// An unknown field is a typo in a security-relevant file — a
	// mistyped "secret" that silently leaves SecretKey empty would be
	// caught below, but a mistyped tenant id would not.
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("tenant: parse %s: %w", path, err)
	}
	return New(f.Tenants...)
}

// New builds a registry, refusing anything that would make two tenants
// indistinguishable.
//
// A duplicate secret is the one that matters: two tenants sharing it
// means either can act as the other, and the failure is silent because
// both requests authenticate. It is rejected at startup rather than
// discovered in an archive.
func New(tenants ...Tenant) (*Registry, error) {
	if len(tenants) == 0 {
		return nil, fmt.Errorf("tenant: registry is empty; a service that speaks " +
			"for nobody would accept nobody")
	}
	r := &Registry{all: make([]Tenant, 0, len(tenants)), bySite: map[string]*Tenant{}}
	// Collision sets, not the lookup map: bySite is only populated once
	// every entry has been accepted, so checking it here would check an
	// empty map and let a duplicate site_key through. It did.
	seenID, seenSite, seenSecret := map[string]bool{}, map[string]string{}, map[string]bool{}

	for _, t := range tenants {
		switch {
		case t.ID == "":
			return nil, fmt.Errorf("tenant: an entry has no id")
		case t.SiteKey == "":
			return nil, fmt.Errorf("tenant %s: no site_key", t.ID)
		case t.SecretKey == "":
			return nil, fmt.Errorf("tenant %s: no secret_key", t.ID)
		case seenID[t.ID]:
			return nil, fmt.Errorf("tenant %s: declared twice", t.ID)
		case seenSite[t.SiteKey] != "":
			return nil, fmt.Errorf("tenant %s: site_key already belongs to %s",
				t.ID, seenSite[t.SiteKey])
		case seenSecret[t.SecretKey]:
			return nil, fmt.Errorf("tenant %s: secret_key is shared with another tenant; "+
				"either could then act as the other, and both requests would authenticate", t.ID)
		}
		seenID[t.ID], seenSecret[t.SecretKey] = true, true
		seenSite[t.SiteKey] = t.ID
		r.all = append(r.all, t)
	}
	for i := range r.all {
		r.bySite[r.all[i].SiteKey] = &r.all[i]
	}
	return r, nil
}

// BySiteKey resolves the public key a page carries. An ordinary map
// lookup: site_key is public, so there is nothing here to leak.
func (r *Registry) BySiteKey(k string) (*Tenant, bool) {
	t, ok := r.bySite[k]
	return t, ok
}

// BySecretKey resolves the credential an application server presents.
//
// Every entry is compared, with no early return, and each comparison is
// constant-time. A map lookup would key on the secret itself and a loop
// that stopped at the first match would take longer for a tenant later
// in the file — both leak, one about the secret's bytes and one about
// which tenant matched. The registry is small enough that scanning it
// costs nothing worth having back.
func (r *Registry) BySecretKey(k string) (*Tenant, bool) {
	var found *Tenant
	key := []byte(k)
	for i := range r.all {
		if subtle.ConstantTimeCompare(key, []byte(r.all[i].SecretKey)) == 1 {
			found = &r.all[i]
		}
	}
	return found, found != nil
}

// Fingerprint identifies the tenant SET this process serves.
//
// Two services that disagree about who exists will each behave
// correctly on its own and wrongly together: a session opened for a
// tenant the engine has never heard of gets a decision attributed to
// nobody, and no request fails on the way. The fingerprint makes that
// disagreement visible from outside, which a log line cannot be.
//
// IT COVERS IDS AND SITE KEYS, NOT SECRETS. Both are public — a site
// key is in the page source — so publishing a hash of them costs
// nothing. Hashing the secrets would be a hash of credentials on an
// unauthenticated endpoint, and its safety would depend entirely on
// their entropy, which this package cannot know.
//
// Credential divergence is caught anyway, and more directly: the same
// application key is presented to both services in `make shadow-http`,
// so a mismatch is a 401 rather than a subtle wrong answer.
func (r *Registry) Fingerprint() string {
	rows := make([]string, 0, len(r.all))
	for _, t := range r.all {
		rows = append(rows, t.ID+"\x00"+t.SiteKey)
	}
	sort.Strings(rows)
	sum := sha256.Sum256([]byte(strings.Join(rows, "\x1e")))
	return hex.EncodeToString(sum[:8])
}

// Len reports how many tenants are registered.
func (r *Registry) Len() int { return len(r.all) }

// IDs lists the tenant ids, for logging what a process is serving.
// Secrets are never included.
func (r *Registry) IDs() []string {
	out := make([]string, 0, len(r.all))
	for _, t := range r.all {
		out = append(out, t.ID)
	}
	return out
}
