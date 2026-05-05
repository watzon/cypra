# ADR-0013: Postgres Role Separation

Status: proposed

Cypra will separate runtime and migration database roles, including narrow audit-log grants and CI checks that runtime cannot delete audit entries.
