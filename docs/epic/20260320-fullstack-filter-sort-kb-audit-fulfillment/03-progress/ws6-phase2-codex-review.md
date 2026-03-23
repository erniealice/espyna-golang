# WS6 Phase 2 Codex Review

## Findings
1. High: `ListByEntity` requires `workspace_id` in both query paths, but `LogEntry` never inserts it (`contrib/postgres/internal/adapter/audit/audit_adapter.go:74-96` vs `:165-187`). If the column is required, writes fail; if it defaults/nullable, workspace-scoped reads will miss freshly written audit rows.
2. High: Cursor pagination loses sub-second precision. Entries are converted to `time.RFC3339` on read and cursor generation (`contrib/postgres/internal/adapter/audit/audit_adapter.go:206,247-249`), then reused in `(occurred_at, id) < ($4, $5)`. Rows in the same second can be skipped across pages.
3. Medium: `DiffAndLog` only iterates `req.NewData` for UPDATE (`contrib/postgres/internal/adapter/audit/diff.go:52-69`). Fields present in `OldData` but omitted from `NewData` are never logged as removals, so UPDATE diffs are incomplete for unset/delete-key cases.
4. Medium: `ListByEntity` loads field changes with one query per returned entry (`contrib/postgres/internal/adapter/audit/audit_adapter.go:218-241`). That is an N+1 pattern and will scale worse than a batched `WHERE audit_entry_id = ANY(...)` fetch.

## Checklist
- Direct SQL only: Yes. The adapter uses raw constant SQL plus `QueryRowContext`/`ExecContext`; no `PostgresOperations.Create` usage found.
- Caller transaction via `getExecutor(ctx)`: Yes. It reads the transaction from context and uses `GetTx()` when available (`contrib/postgres/internal/adapter/audit/audit_adapter.go:38-55`).
- Cursor pagination: Partially. It uses `LIMIT+1` and Base64-encoded JSON cursors, but the timestamp-precision bug above makes the keyset cursor unsafe.
- INSERT/UPDATE/DELETE diff branches: INSERT and DELETE are handled; UPDATE misses removed keys.
- Excluded fields filtering: Yes for exact field-name matches in all three branches (`contrib/postgres/internal/adapter/audit/diff.go:40-42,54-56,73-75`; `contrib/postgres/internal/adapter/audit/excluded_fields.go:8-25`).
- Middleware actor extraction: Safe from panics. It uses a string type assertion and falls back to `system` when `uid` is absent (`internal/infrastructure/adapters/primary/http/vanilla/middleware/audit_context.go:20-25`). Minor caveat: `RemoteAddr` trimming is IPv4-oriented and can mis-handle IPv6 literals (`.../audit_context.go:36-40`).
- SQL injection risk: None seen. Query text is static and all variable values are parameterized.
