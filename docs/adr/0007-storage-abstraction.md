# ADR-0007: Storage Abstraction

Status: accepted

Cypra will use an internal storage interface with local-disk HMAC signed URLs and S3-compatible presigned URLs behind bounded TTLs.

## Context

Cypra stores tenant-owned binary objects such as profile pictures and future export artifacts. Operators need a development-friendly backend that works from a single local binary, while production installs need S3-compatible object storage such as AWS S3, MinIO, R2, or equivalent providers.

Storage URLs must not expose raw filesystem paths or long-lived bucket credentials. The application also needs a narrow interface so later handlers can record metadata in `storage_objects` without coupling business logic to one storage backend.

## Decision

`internal/storage` defines a small `Store` interface: `Put`, `Get`, `Delete`, and `SignedURL(key, ttl)`. Implementations cap signed URL TTLs at one hour.

The `local-disk` backend stores bytes under an operator-provided root and returns proxy URLs of the form `/storage/<payload>.<sig>`. The payload contains the object key and expiry. The signature is HMAC-SHA256 over the base64url payload, using an operator-provided secret.

The `s3-compatible` backend uses AWS SDK v2 and `s3.PresignClient.PresignGetObject`. It supports custom endpoints and path-style addressing for MinIO/R2-compatible providers.

## Consequences

- Storage handlers can swap backends without changing auth/dashboard code.
- Local development does not require a bucket or cloud credentials.
- Production object reads use native S3-compatible presigning instead of Cypra proxying every byte.
- Operators are responsible for keeping the local-disk root and signing secret private.
