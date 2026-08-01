//go:build postgresql

package postgres

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"unicode"

	"github.com/lib/pq"
)

// =============================================================================
// A reference libpq keyword/value parser
// =============================================================================
//
// parseKeywordValueDSN mirrors the keyword/value conninfo grammar
// (https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING-KEYWORD-VALUE):
// keywords and values are separated by '=' with optional surrounding
// whitespace; a value may be single-quoted; inside or outside quotes a
// backslash escapes the next character. Round-tripping buildDSN through this
// parser is how the escaping tests below assert that no interpolated value can
// terminate its own field or start a new keyword.
//
// Whitespace here is unicode.IsSpace, NOT the ASCII set. That deliberately
// models the parser that actually runs — lib/pq's parseOpts terminates keywords
// and unquoted values on unicode.IsSpace — and it is strictly the more
// aggressive of the two candidate parsers (C libpq's isspace() splits on a
// subset). An ASCII-only reference parser is a false gate: it round-trips a
// value containing U+00A0 that the real driver would split into two fields.
// TestBuildDSNIsInertToTheRealDriver below re-checks the same vectors against
// lib/pq itself so this parser can never be the only witness.
func parseKeywordValueDSN(dsn string) (map[string]string, []string, error) {
	out := map[string]string{}
	var order []string
	r := []rune(dsn)
	i := 0

	skipSpace := func() {
		for i < len(r) && unicode.IsSpace(r[i]) {
			i++
		}
	}

	for {
		skipSpace()
		if i >= len(r) {
			return out, order, nil
		}

		var keyword strings.Builder
		for i < len(r) && r[i] != '=' && !unicode.IsSpace(r[i]) {
			keyword.WriteRune(r[i])
			i++
		}
		skipSpace()
		if i >= len(r) || r[i] != '=' {
			return nil, nil, fmt.Errorf("missing '=' after keyword %q at offset %d", keyword.String(), i)
		}
		i++ // consume '='
		skipSpace()

		var value strings.Builder
		if i < len(r) && r[i] == '\'' {
			i++
			closed := false
			for i < len(r) {
				switch r[i] {
				case '\\':
					i++
					if i >= len(r) {
						return nil, nil, fmt.Errorf("dangling escape inside quoted value for %q", keyword.String())
					}
					value.WriteRune(r[i])
					i++
				case '\'':
					i++
					closed = true
				default:
					value.WriteRune(r[i])
					i++
				}
				if closed {
					break
				}
			}
			if !closed {
				return nil, nil, fmt.Errorf("unterminated quoted value for %q", keyword.String())
			}
		} else {
			for i < len(r) {
				c := r[i]
				if unicode.IsSpace(c) {
					break
				}
				if c == '\\' {
					i++
					if i >= len(r) {
						return nil, nil, fmt.Errorf("dangling escape in value for %q", keyword.String())
					}
					value.WriteRune(r[i])
					i++
					continue
				}
				value.WriteRune(c)
				i++
			}
		}

		key := keyword.String()
		if key == "" {
			return nil, nil, fmt.Errorf("empty keyword at offset %d", i)
		}
		if _, dup := out[key]; !dup {
			order = append(order, key)
		}
		out[key] = value.String()
	}
}

func TestReferenceParserHandlesLibpqQuoting(t *testing.T) {
	cases := []struct {
		dsn  string
		want map[string]string
	}{
		{`host=localhost port=5432`, map[string]string{"host": "localhost", "port": "5432"}},
		{`host='local host'`, map[string]string{"host": "local host"}},
		{`password=''`, map[string]string{"password": ""}},
		{`password='a\'b'`, map[string]string{"password": "a'b"}},
		{`password=a\\b`, map[string]string{"password": `a\b`}},
		{`password='a\\b c'`, map[string]string{"password": `a\b c`}},
		{`host = localhost`, map[string]string{"host": "localhost"}},
		{`options='-c statement_timeout=30000 -c lock_timeout=10000'`, map[string]string{"options": "-c statement_timeout=30000 -c lock_timeout=10000"}},
	}
	for _, tc := range cases {
		got, _, err := parseKeywordValueDSN(tc.dsn)
		if err != nil {
			t.Errorf("parse(%q): %v", tc.dsn, err)
			continue
		}
		if len(got) != len(tc.want) {
			t.Errorf("parse(%q) = %v, want %v", tc.dsn, got, tc.want)
			continue
		}
		for k, v := range tc.want {
			if got[k] != v {
				t.Errorf("parse(%q)[%q] = %q, want %q", tc.dsn, k, got[k], v)
			}
		}
	}
}

// TestReferenceParserSplitsOnUnicodeWhitespace guards the reference parser
// itself. If it ever drifts back to an ASCII-only whitespace set, every
// round-trip assertion in this file silently becomes a false gate: an injected
// keyword separated by U+00A0 would "round trip" here while the real driver
// splits it into two fields. The expectations below are lib/pq's behaviour.
func TestReferenceParserSplitsOnUnicodeWhitespace(t *testing.T) {
	for _, sep := range unicodeSpaceSeparators {
		fields, _, err := parseKeywordValueDSN("host=db.internal" + sep.value + "client_encoding=LATIN1")
		if err != nil {
			t.Errorf("%s: parse error: %v", sep.name, err)
			continue
		}
		if len(fields) != 2 || fields["host"] != "db.internal" || fields["client_encoding"] != "LATIN1" {
			t.Errorf("%s: reference parser did not split an unquoted value on %s: %v — it is ASCII-only again",
				sep.name, sep.name, fields)
		}
	}
}

// unicodeSpaceSeparators are the runes lib/pq's parseOpts treats as a field
// separator (unicode.IsSpace) beyond the plain ASCII space. Each one is an
// injection vector against an ASCII-only quoting predicate.
var unicodeSpaceSeparators = []struct {
	name  string
	value string
}{
	{"ascii space", " "},
	{"tab", "\t"},
	{"newline", "\n"},
	{"NBSP U+00A0", "\u00a0"},
	{"NEL U+0085", "\u0085"},
	{"ogham space U+1680", "\u1680"},
	{"en quad U+2000", "\u2000"},
	{"hair space U+200A", "\u200a"},
	{"line separator U+2028", "\u2028"},
	{"paragraph separator U+2029", "\u2029"},
	{"narrow NBSP U+202F", "\u202f"},
	{"medium math space U+205F", "\u205f"},
	{"ideographic space U+3000", "\u3000"},
}

// =============================================================================
// pgKV: exact rendered form
// =============================================================================

func TestPgKVRendersExactEscapedForm(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  string
	}{
		{"plain", "localhost", `host=localhost`},
		{"digits", "5432", `host=5432`},
		{"empty is quoted", "", `host=''`},
		{"space", "local host", `host='local host'`},
		{"leading space", " localhost", `host=' localhost'`},
		{"trailing space", "localhost ", `host='localhost '`},
		{"tab", "a\tb", "host='a\tb'"},
		{"newline", "a\nb", "host='a\nb'"},
		{"carriage return", "a\rb", "host='a\rb'"},
		{"vertical tab", "a\vb", "host='a\vb'"},
		{"form feed", "a\fb", "host='a\fb'"},
		{"single quote", "a'b", `host='a\'b'`},
		{"only a quote", "'", `host='\''`},
		{"backslash", `a\b`, `host=a\\b`},
		{"trailing backslash", `ab\`, `host=ab\\`},
		{"backslash and space", `a\b c`, `host='a\\b c'`},
		{"backslash and quote", `a\'b`, `host='a\\\'b'`},
		{"equals sign", "a=b", `host=a=b`},
		{"injected keyword", "localhost sslmode=require", `host='localhost sslmode=require'`},
		{"injected keyword via quote", `x' sslmode='require`, `host='x\' sslmode=\'require'`},
		{"unicode", "héllo", `host=héllo`},

		// Unicode whitespace: lib/pq's parseOpts terminates an unquoted value on
		// unicode.IsSpace, so every one of these MUST be quoted. An ASCII-only
		// predicate emits them bare and the value silently continues as further
		// keyword=value pairs (last-wins), which is a live sslmode-downgrade /
		// options-blanking vector.
		{"nbsp U+00A0", "a\u00a0b", "host='a\u00a0b'"},
		{"next line U+0085", "a\u0085b", "host='a\u0085b'"},
		{"ogham space U+1680", "a\u1680b", "host='a\u1680b'"},
		{"en quad U+2000", "a\u2000b", "host='a\u2000b'"},
		{"hair space U+200A", "a\u200ab", "host='a\u200ab'"},
		{"line separator U+2028", "a\u2028b", "host='a\u2028b'"},
		{"paragraph separator U+2029", "a\u2029b", "host='a\u2029b'"},
		{"narrow nbsp U+202F", "a\u202fb", "host='a\u202fb'"},
		{"medium math space U+205F", "a\u205fb", "host='a\u205fb'"},
		{"ideographic space U+3000", "a\u3000b", "host='a\u3000b'"},
		{"nbsp keyword injection", "pw\u00a0sslmode=disable", "host='pw\u00a0sslmode=disable'"},
		{"ideographic keyword injection", "db.internal\u3000client_encoding=LATIN1", "host='db.internal\u3000client_encoding=LATIN1'"},

		// Not whitespace to unicode.IsSpace and not to lib/pq either: a
		// zero-width space is Cf, not Z, so it stays inside an unquoted value.
		// Pinned so the predicate is not "quote on anything non-ASCII".
		{"zero width space U+200B is not whitespace", "a\u200bb", "host=a\u200bb"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pgKV("host", tc.value); got != tc.want {
				t.Errorf("pgKV(host, %q) = %q, want %q", tc.value, got, tc.want)
			}
			// Whatever the rendering, libpq must read back the original value.
			parsed, _, err := parseKeywordValueDSN(pgKV("host", tc.value))
			if err != nil {
				t.Fatalf("rendered form does not parse: %v", err)
			}
			if len(parsed) != 1 {
				t.Fatalf("value produced %d fields, want 1: %v", len(parsed), parsed)
			}
			if parsed["host"] != tc.value {
				t.Errorf("round trip = %q, want %q", parsed["host"], tc.value)
			}
		})
	}
}

// =============================================================================
// buildDSN: full-string round trips
// =============================================================================

// dsnHostile is a value that would break or hijack an unescaped DSN: it carries
// a space, a single quote, a backslash and a complete keyword/value pair.
const dsnHostile = `p a\ss' sslmode=require host=evil.example.com '`

// dsnUnicodeHostile is the same attack expressed with NON-ASCII whitespace: an
// ASCII-only quoting predicate emits it unquoted, lib/pq ends the value at the
// U+00A0 and parses the rest as further keyword=value pairs (last wins), which
// downgrades sslmode and blanks the options payload carrying the timeouts.
const dsnUnicodeHostile = "pw\u00a0sslmode=disable\u3000client_encoding=LATIN1\u2028host=evil.example.com"

func TestBuildDSNRoundTripsHostileValues(t *testing.T) {
	cases := []struct {
		name string
		cfg  *PostgresConfig
	}{
		{
			name: "hostile password",
			cfg: &PostgresConfig{
				Host: "db.internal", Port: "5432", Name: "app", User: "app",
				Password: dsnHostile, SSLMode: "require",
			},
		},
		{
			name: "hostile host",
			cfg: &PostgresConfig{
				Host: dsnHostile, Port: "5432", Name: "app", User: "app",
				Password: "pw", SSLMode: "require",
			},
		},
		{
			name: "hostile everything",
			cfg: &PostgresConfig{
				Host: dsnHostile, Port: `54 32`, Name: `db'name`, User: `us\er`,
				Password: dsnHostile, SSLMode: `require' options='-c statement_timeout=0`,
			},
		},
		{
			name: "spaces everywhere",
			cfg: &PostgresConfig{
				Host: "a b", Port: "c d", Name: "e f", User: "g h",
				Password: "i j", SSLMode: "k l",
			},
		},
		{
			name: "unicode whitespace injection in password",
			cfg: &PostgresConfig{
				Host: "db.internal", Port: "5432", Name: "app", User: "app",
				Password: dsnUnicodeHostile, SSLMode: "require",
				StatementTimeout: timeoutFromSeconds(30), LockTimeout: timeoutFromSeconds(10), IdleTxTimeout: timeoutFromSeconds(60),
			},
		},
		{
			name: "unicode whitespace injection in host",
			cfg: &PostgresConfig{
				Host: dsnUnicodeHostile, Port: "5432", Name: "app", User: "app",
				Password: "pw", SSLMode: "require",
			},
		},
		{
			name: "unicode whitespace injection everywhere",
			cfg: &PostgresConfig{
				Host: dsnUnicodeHostile, Port: "54\u200532", Name: "db\u2028name", User: "us\u3000er",
				Password: dsnUnicodeHostile, SSLMode: "require\u00a0options=",
			},
		},
		{
			name: "empty optional values",
			cfg: &PostgresConfig{
				Host: "", Port: "", Name: "", User: "", Password: "", SSLMode: "",
			},
		},
		{
			name: "timeouts present",
			cfg: &PostgresConfig{
				Host: "h", Port: "5432", Name: "d", User: "u", Password: "p", SSLMode: "disable",
				StatementTimeout: timeoutFromSeconds(30), LockTimeout: timeoutFromSeconds(10), IdleTxTimeout: timeoutFromSeconds(60),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dsn := buildDSN(tc.cfg)
			fields, order, err := parseKeywordValueDSN(dsn)
			if err != nil {
				t.Fatalf("buildDSN produced an unparseable string: %v\n%s", err, dsn)
			}

			want := map[string]string{
				"host":            tc.cfg.Host,
				"port":            tc.cfg.Port,
				"dbname":          tc.cfg.Name,
				"user":            tc.cfg.User,
				"sslmode":         tc.cfg.SSLMode,
				"connect_timeout": "5",
			}
			if opts := sessionOptions(tc.cfg); opts != "" {
				want["options"] = opts
			}
			if tc.cfg.Password != "" {
				want["password"] = tc.cfg.Password
			}

			if len(fields) != len(want) {
				t.Errorf("DSN carries %d fields %v, want %d %v\n%s", len(fields), order, len(want), want, dsn)
			}
			for k, v := range want {
				got, ok := fields[k]
				if !ok {
					t.Errorf("DSN is missing %q\n%s", k, dsn)
					continue
				}
				if got != v {
					t.Errorf("DSN %q = %q, want %q\n%s", k, got, v, dsn)
				}
			}
			for k := range fields {
				if _, expected := want[k]; !expected {
					t.Errorf("DSN gained an unexpected keyword %q — a value escaped its field\n%s", k, dsn)
				}
			}
		})
	}
}

func TestBuildDSNEmptySSLModeIsNotDefaultedHere(t *testing.T) {
	// resolvePostgresConfig owns the sslmode default; buildDSN must render an
	// empty value faithfully rather than quietly substituting one.
	dsn := buildDSN(&PostgresConfig{Host: "h", Port: "1", Name: "d", User: "u", SSLMode: ""})
	if !strings.Contains(dsn, `sslmode=''`) {
		t.Errorf("empty sslmode not rendered as an empty quoted value: %s", dsn)
	}
}

func TestBuildDSNOmitsEmptyPassword(t *testing.T) {
	dsn := buildDSN(&PostgresConfig{Host: "h", Port: "1", Name: "d", User: "u", SSLMode: "disable"})
	if strings.Contains(dsn, "password") {
		t.Errorf("empty password should be omitted entirely: %s", dsn)
	}
	withPassword := buildDSN(&PostgresConfig{Host: "h", Port: "1", Name: "d", User: "u", SSLMode: "disable", Password: "pw"})
	if !strings.Contains(withPassword, "password=pw") {
		t.Errorf("password not rendered: %s", withPassword)
	}
}

// =============================================================================
// sessionOptions: unit conversion and the 0-disables rule
// =============================================================================

func TestSessionOptions(t *testing.T) {
	cases := []struct {
		name      string
		statement int
		lock      int
		idleTx    int
		want      string
	}{
		{
			name: "all three", statement: 30, lock: 10, idleTx: 60,
			want: "-c statement_timeout=30000 -c idle_in_transaction_session_timeout=60000 -c lock_timeout=10000",
		},
		{name: "all disabled", want: ""},
		{name: "statement only", statement: 30, want: "-c statement_timeout=30000"},
		{name: "lock only", lock: 10, want: "-c lock_timeout=10000"},
		{name: "idle-tx only", idleTx: 60, want: "-c idle_in_transaction_session_timeout=60000"},
		{
			name: "statement disabled, others on", lock: 10, idleTx: 60,
			want: "-c idle_in_transaction_session_timeout=60000 -c lock_timeout=10000",
		},
		{name: "one second", statement: 1, want: "-c statement_timeout=1000"},
		{name: "ceiling", statement: maxTimeoutSeconds, want: "-c statement_timeout=3600000"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sessionOptions(&PostgresConfig{
				StatementTimeout: timeoutFromSeconds(tc.statement),
				LockTimeout:      timeoutFromSeconds(tc.lock),
				IdleTxTimeout:    timeoutFromSeconds(tc.idleTx),
			})
			if got != tc.want {
				t.Errorf("sessionOptions = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBuildDSNDropsOptionsWhenAllTimeoutsDisabled(t *testing.T) {
	dsn := buildDSN(&PostgresConfig{Host: "h", Port: "1", Name: "d", User: "u", SSLMode: "disable"})
	if strings.Contains(dsn, "options") {
		t.Errorf("options keyword should be dropped when every timeout is disabled: %s", dsn)
	}
	fields, _, err := parseKeywordValueDSN(dsn)
	if err != nil {
		t.Fatalf("unparseable: %v", err)
	}
	if _, ok := fields["options"]; ok {
		t.Errorf("options present: %v", fields)
	}
}

func TestBuildDSNCarriesOptionsAsOneValue(t *testing.T) {
	dsn := buildDSN(&PostgresConfig{
		Host: "h", Port: "1", Name: "d", User: "u", SSLMode: "disable",
		StatementTimeout: timeoutFromSeconds(30), LockTimeout: timeoutFromSeconds(10), IdleTxTimeout: timeoutFromSeconds(60),
	})
	if !strings.Contains(dsn, `options='-c statement_timeout=30000 -c idle_in_transaction_session_timeout=60000 -c lock_timeout=10000'`) {
		t.Errorf("options payload not quoted as a single value: %s", dsn)
	}
	fields, _, err := parseKeywordValueDSN(dsn)
	if err != nil {
		t.Fatalf("unparseable: %v", err)
	}
	if got := fields["options"]; got != "-c statement_timeout=30000 -c idle_in_transaction_session_timeout=60000 -c lock_timeout=10000" {
		t.Errorf("options = %q", got)
	}
	if _, leaked := fields["statement_timeout"]; leaked {
		t.Errorf("an option leaked out as its own keyword: %v", fields)
	}
}

func TestSecondsLabel(t *testing.T) {
	cases := []struct {
		name string
		in   resolvedTimeout
		want string
	}{
		{"explicit disabled", timeoutFromSeconds(0), "disabled"},
		{"zero value fails safe", resolvedTimeout{}, "disabled"},
		{"one second", timeoutFromSeconds(1), "1s"},
		{"thirty seconds", timeoutFromSeconds(30), "30s"},
		{"ceiling", timeoutFromSeconds(maxTimeoutSeconds), "3600s"},
	}
	for _, tc := range cases {
		if got := secondsLabel(tc.in); got != tc.want {
			t.Errorf("%s: secondsLabel = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// =============================================================================
// Driver truth: the real lib/pq parser, not our reference implementation
// =============================================================================

// pqEnvIsolate removes every PG* environment variable for the duration of the
// test. lib/pq folds them into the parsed option set (and panics outright on a
// few of them), so leaving them in place would make a driver-truth assertion
// depend on the operator's shell.
func pqEnvIsolate(t *testing.T) {
	t.Helper()
	for _, kv := range os.Environ() {
		k, v, _ := strings.Cut(kv, "=")
		if !strings.HasPrefix(k, "PG") {
			continue
		}
		if err := os.Unsetenv(k); err != nil {
			t.Fatalf("unset %s: %v", k, err)
		}
		t.Cleanup(func() { _ = os.Setenv(k, v) })
	}
}

// TestBuildDSNIsInertToTheRealDriver runs the injection vectors through
// github.com/lib/pq itself — the parser that actually consumes this DSN at
// runtime — so the escaping claim never rests on the reference parser above.
//
// client_encoding is the observable: pq.NewConnector rejects any value other
// than UTF8 while parsing (connector.go), and it does so without opening a
// socket. So "the built DSN parses cleanly" == "the injected
// client_encoding=LATIN1 never became a keyword of its own".
func TestBuildDSNIsInertToTheRealDriver(t *testing.T) {
	pqEnvIsolate(t)

	const payload = "client_encoding=LATIN1"

	base := func() *PostgresConfig {
		return &PostgresConfig{
			Host: "db.internal", Port: "5432", Name: "app", User: "app",
			Password: "pw", SSLMode: "require",
			StatementTimeout: timeoutFromSeconds(30), LockTimeout: timeoutFromSeconds(10), IdleTxTimeout: timeoutFromSeconds(60),
		}
	}

	for _, sep := range unicodeSpaceSeparators {
		for _, field := range []string{"password", "host", "dbname", "user"} {
			t.Run(sep.name+"/"+field, func(t *testing.T) {
				injected := "v" + sep.value + payload
				cfg := base()
				switch field {
				case "password":
					cfg.Password = injected
				case "host":
					cfg.Host = injected
				case "dbname":
					cfg.Name = injected
				case "user":
					cfg.User = injected
				}
				dsn := buildDSN(cfg)
				if _, err := pq.NewConnector(dsn); err != nil {
					t.Errorf("lib/pq rejected the built DSN (%v) — the %s value escaped its field\n%q", err, field, dsn)
				}
			})
		}
	}

	// Control: the same payload interpolated WITHOUT pgKV must be caught by the
	// driver. If this stops failing, the assertions above prove nothing.
	t.Run("control/unescaped payload is detected", func(t *testing.T) {
		for _, sep := range unicodeSpaceSeparators {
			unescaped := "host=db.internal port=5432 dbname=app user=app sslmode=require password=pw" + sep.value + payload
			if _, err := pq.NewConnector(unescaped); err == nil {
				t.Errorf("%s: an unescaped injection went undetected — this test cannot witness injection", sep.name)
			}
		}
	})
}

// TestBuildDSNKeepsSSLModeAndOptionsUnderInjection is the impact assertion
// behind the escaping rule: the two fields an injected keyword would target are
// sslmode (TLS downgrade) and options (the three session timeouts this adapter
// exists to set). password is rendered LAST, so an unescaped password payload
// would override both under libpq last-wins semantics.
func TestBuildDSNKeepsSSLModeAndOptionsUnderInjection(t *testing.T) {
	wantOptions := "-c statement_timeout=30000 -c idle_in_transaction_session_timeout=60000 -c lock_timeout=10000"

	for _, sep := range unicodeSpaceSeparators {
		cfg := &PostgresConfig{
			Host: "db.internal", Port: "5432", Name: "app", User: "app",
			SSLMode:          "require",
			Password:         "pw" + sep.value + "sslmode=disable" + sep.value + "options=",
			StatementTimeout: timeoutFromSeconds(30), LockTimeout: timeoutFromSeconds(10), IdleTxTimeout: timeoutFromSeconds(60),
		}
		dsn := buildDSN(cfg)
		fields, _, err := parseKeywordValueDSN(dsn)
		if err != nil {
			t.Errorf("%s: unparseable DSN: %v\n%s", sep.name, err, dsn)
			continue
		}
		if got := fields["sslmode"]; got != "require" {
			t.Errorf("%s: sslmode downgraded to %q by a password payload\n%s", sep.name, got, dsn)
		}
		if got := fields["options"]; got != wantOptions {
			t.Errorf("%s: options payload clobbered: %q\n%s", sep.name, got, dsn)
		}
		if got := fields["password"]; got != cfg.Password {
			t.Errorf("%s: password did not round trip: %q\n%s", sep.name, got, dsn)
		}
	}
}
