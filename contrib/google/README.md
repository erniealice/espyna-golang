# Google contrib providers

This module contains the build-tagged Google adapters. Resource selection and runtime identity are
separate axes:

| Concern | Resource selector | Credential selector |
|---|---|---|
| Firebase Auth | `AUTH_FIREBASE_PROJECT_ID` | ADC when `AUTH_FIREBASE_CREDENTIALS_FILE` is absent; that scoped file only for local/non-managed use |
| GCS | `STORAGE_GCS_PROJECT_ID` + physical `STORAGE_GCS_BUCKET` | ADC when `STORAGE_GCS_CREDENTIALS_FILE` is absent; that scoped file only for local/non-managed use |

Credential metadata never supplies a target project. Inline `*_SA_*`, `*_USE_SERVICE_ACCOUNT`, and
`*_SERVICE_ACCOUNT_KEY_PATH` names are retired and rejected by name. The package never writes the
process-global `GOOGLE_APPLICATION_CREDENTIALS`.

On Cloud Run, attach a dedicated runtime service account owned by the Run project. The same principal
may receive cross-project access without moving the account:

- `roles/firebaseauth.admin` in the Firebase resource project because the adapter performs Auth
  lifecycle reads and writes;
- `roles/storage.objectCreator` plus `roles/storage.objectViewer` on the configured bucket only;
- `roles/secretmanager.secretAccessor` on each named deployed secret only.

Do not grant basic roles, a `*ServiceAgent` role, project-wide Secret Accessor, Storage Object User,
Object Admin, Storage Admin, or bucket administration. The deployer separately needs authority to
attach the runtime identity (`iam.serviceAccounts.actAs`); that is not a runtime permission.

`container_name` is a physical bucket/container. Feature namespace belongs in `object_key` (for
example `templates/outcome_matrix/<id>.docx`). Composition may replace a local/mock fallback with a
configured cloud default only while creating a new locator. Reads pass a persisted locator verbatim.
Cloud buckets are pre-provisioned; the GCS health probe uses object listing and does not require
bucket metadata or creation rights.

Focused verification:

```sh
go test -short -count=1 -tags='firebase gcs google' \
  ./consumer \
  ./contrib/google/internal/common/gcp \
  ./contrib/google/internal/auth/firebase \
  ./contrib/google/internal/storage/gcs
```

Tests and source validation are read-only. Creating accounts, changing IAM, uploading objects, or
deploying revisions is an external mutation and requires explicit authority, exact target resolution,
current policy/traffic snapshots, and a rollback command.
