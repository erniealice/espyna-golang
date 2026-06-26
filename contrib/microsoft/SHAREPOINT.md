# Future SharePoint STORAGE adapter (delineation note)

This module's `internal/common` package is a **concern-agnostic Azure-AD app-token
primitive**. It hardcodes no concern prefix and holds no package-global token
state. Each concern owns its `{CONCERN}_{PROVIDER}_*` env vars and constructs its
**own** `*common.Client` instance (own credentials, own token cache, own mutex).

This file records the shape of a future **SharePoint STORAGE** adapter so the
delineation is explicit. **The adapter does not exist yet — this is documentation
only, not an implementation.**

## Naming convention

`{CONCERN}_{PROVIDER}_{FIELD}` — the concern owns the prefix, `common` only knows
`{FIELD}` (`TENANT_ID`, `CLIENT_ID`, `CLIENT_SECRET`, `TIMEOUT`).

| Concern         | Provider     | Prefix                  | Location                          | Status   |
| --------------- | ------------ | ----------------------- | --------------------------------- | -------- |
| EMAIL (Graph)   | `microsoft`  | `EMAIL_MICROSOFT_`      | `internal/email/`                 | LANDED   |
| STORAGE         | `sharepoint` | `STORAGE_SHAREPOINT_`   | `internal/storage/sharepoint/`    | FUTURE   |

## EMAIL (today)

```go
// internal/email/adapter.go
client, _ := common.NewClient(common.FromEnv("EMAIL_MICROSOFT_"))
```

Reads `EMAIL_MICROSOFT_{TENANT_ID,CLIENT_ID,CLIENT_SECRET,DELEGATE_EMAIL,
FROM_EMAIL,FROM_NAME,REDIRECT_URL,ACCESS_TOKEN,REFRESH_TOKEN,TOKEN_TYPE,
TOKEN_EXPIRY,TIMEOUT}`.

## STORAGE / SharePoint (future)

When built, the SharePoint storage adapter will live at
`internal/storage/sharepoint/` and construct its **own** Azure-AD token client:

```go
// internal/storage/sharepoint/adapter.go (does not exist yet)
client, _ := common.NewClient(common.FromEnv("STORAGE_SHAREPOINT_"))
```

It will read `STORAGE_SHAREPOINT_{TENANT_ID,CLIENT_ID,CLIENT_SECRET,SITE_URL,...}`.

## Why two independent clients

`STORAGE_SHAREPOINT_*` and `EMAIL_MICROSOFT_*` are **independent Azure apps**.
Because `common.Client` is per-instance (no package singleton), both concerns can
run in the same process at the same time, each holding a different Azure app and
a separate token cache. Adding the storage concern requires **no change** to
`internal/common` — it only injects a different prefix into `common.FromEnv` and
holds its own `*common.Client`.
