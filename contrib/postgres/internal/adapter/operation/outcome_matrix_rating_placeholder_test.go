//go:build postgresql

package operation

import (
	"testing"

	"github.com/erniealice/espyna-golang/shared/placeholder"
)

// TestPlaceholderRegistry_EveryAllowlistedTagHasLoaderMapping is the pairing
// guard of schema-proposal.md §10: adding a tag to the shared/placeholder
// registry without a field→column mapping in this adapter's batched loader
// would make every cell carrying it PLACEHOLDER_UNRESOLVED at save time.
func TestPlaceholderRegistry_EveryAllowlistedTagHasLoaderMapping(t *testing.T) {
	for _, tag := range placeholder.Allowed() {
		cols, ok := placeholderLoaderRoots[placeholder.Root(tag)]
		if !ok {
			t.Errorf("allowlisted tag %q: no loader for root %q", tag, placeholder.Root(tag))
			continue
		}
		if col := cols[tag]; col == "" {
			t.Errorf("allowlisted tag %q: no field→column mapping in the %q loader", tag, placeholder.Root(tag))
		}
	}
	// And the reverse: the loader maps nothing the registry does not allow.
	for root, cols := range placeholderLoaderRoots {
		for tag := range cols {
			if !placeholder.IsAllowed(tag) || placeholder.Root(tag) != root {
				t.Errorf("loader mapping %q (root %q) is not an allowlisted tag of that root", tag, root)
			}
		}
	}
}
