//go:build mock_auth

package mock

import (
	"log"
	"os"
	"strings"
)

// SEC-039: a binary compiled with the `mock_auth` build tag uses
// MockAuthorizationService (AllowAll) — every caller is treated as a superadmin.
// Such a binary must NEVER run against a real (remote) database. This init guard
// fatals at startup when a non-local Postgres host is configured, so an accidental
// `go build -tags mock_auth` (bypassing the build-script case statement) cannot
// be deployed to serve real data.
//
// Local/unset hosts are allowed (tests + local dev). The guard runs at package
// import, before any DB connection is opened.
func init() {
	host := strings.ToLower(strings.TrimSpace(os.Getenv("DATABASE_POSTGRES_HOST")))
	if host == "" {
		host = strings.ToLower(strings.TrimSpace(os.Getenv("POSTGRES_HOST")))
	}
	switch host {
	case "", "localhost", "127.0.0.1", "::1", "host.docker.internal":
		// local / dev / unset — mock auth is acceptable here.
	default:
		log.Fatalf("[SECURITY][SEC-039] binary compiled with -tags mock_auth (AllowAll authorization) "+
			"but DATABASE_POSTGRES_HOST=%q is not local — refusing to start against a real database.", host)
	}
}
