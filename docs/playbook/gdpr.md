# GDPR Notes

Cypra provides export, deletion, redaction, and tenant-cascade mechanics. The operator decides lawful basis, retention policy, and sub-processor disclosures.

User deletion is idempotent: the user row is scrubbed, audit rows are redacted in place, storage objects are marked deleted, and `gdpr_deletions` records completion.
