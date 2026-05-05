# ADR-0009: GORM with Explicit SQL Migrations

Status: proposed

Cypra will use GORM for application queries through `TenantScopedDB` while keeping all DDL in explicit `golang-migrate` SQL files.
