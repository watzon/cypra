# GDPR Notes

Cypra provides export, deletion, redaction, tenant-cascade, and DSR-resurrection guard mechanics. The operator decides lawful basis, retention policy, data-processing terms, and sub-processor disclosures.

## Data Subject Request Checklist

1. Identify the user and tenant under the operator's normal identity-verification process.
2. Export with `GET /api/v1/users/{id}/export` and provide the NDJSON to the requester when appropriate.
3. Delete with `DELETE /api/v1/users/{id}`.
4. Verify the `gdpr_deletions` ledger row is `done`.
5. Verify user PII is scrubbed and prior audit entries contain the GDPR redaction sentinel.
6. Verify profile-picture storage objects are marked deleted.
7. Record the request, action time, operator, and verification evidence outside Cypra if your retention policy requires it.

User deletion is idempotent: the user row is scrubbed, audit rows are redacted in place, storage objects are marked deleted, and `gdpr_deletions` records completion.

## Backup Restore Guard

`cypra import` refuses to resurrect a DSR-deleted user unless `--allow-resurrect` is explicitly passed. Treat `--allow-resurrect` as an incident-level decision: record the legal basis, affected user IDs, backup file, operator, and audit evidence before using it.
