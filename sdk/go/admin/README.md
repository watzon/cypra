# Cypra Admin SDK

Typed Go client for `/api/v1` administration endpoints.

## Usage

```go
client, err := admin.New("https://cypra.example.com", os.Getenv("CYPRA_PAT"))
if err != nil {
    log.Fatal(err)
}

projects, err := client.ListProjects(ctx)
```

The client sends PAT credentials with `Authorization: Bearer <pat>`. Instance-level calls add `X-Cypra-Instance-Admin: true`; tenant-scoped calls add `X-Cypra-Tenant-Role: admin` for local/dev compatibility.

## Typed Errors

```go
if errors.Is(err, cypra.ErrConflict) {
    // Duplicate slug, duplicate client id, etc.
}

var apiErr *cypra.Error
if errors.As(err, &apiErr) {
    log.Printf("cypra code=%s status=%d", apiErr.Code, apiErr.StatusCode)
}
```

## Covered Surfaces

Tenants, projects, users, members, OIDC clients, signing keys, audit export, and instance-admin list/invite/demote helpers are exposed as typed methods.
