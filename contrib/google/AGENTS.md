# Google contrib agent boundary

- Treat `{CONCERN}_{PROVIDER}_{FIELD}` as the only runtime provider naming grammar. Project
  identifiers end in `_PROJECT_ID`; do not add aliases or credential-derived fallbacks.
- Keep Firebase and GCS resource targets independent from each other and from the workload identity's
  owning/Cloud Run project.
- Prefer ADC. A concern-scoped `*_CREDENTIALS_FILE` is the only local/non-managed file escape hatch;
  never restore inline private-key fields or mutate `GOOGLE_APPLICATION_CREDENTIALS`.
- Preserve physical storage semantics: resolve a configured default only for a new write/persisted
  locator. Never translate a stored container on read; namespaces belong in object keys.
- Do not broaden GCS initialization/health to bucket metadata or mutation. The runtime contract is
  object create/read/list.
- IAM and cloud resources are outside this module. Discovery is read-only; any service-account, role,
  bucket, secret, deploy, or data mutation needs explicit authority and rollback evidence.
- Run the focused tagged tests in `README.md`, then the direct app composition/build gates for any
  changed provider contract.
