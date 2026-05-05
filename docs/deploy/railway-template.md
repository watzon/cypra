# Railway One-Click Template

Deployment work is outside the active local-runnable objective, so this phase records the intended template shape without creating or verifying a separate Railway repository.

The future `watzon/cypra-railway-template` repository should include:

- `railway.json` with Cypra and Postgres services.
- `STORAGE_BACKEND=s3-compatible` by default.
- R2/B2 setup instructions for bucket, endpoint, region, access key, and secret key.
- A README that walks bootstrap token redemption, tenant creation, provider configuration, and Next.js OIDC setup.

Live Railway beta deploy verification remains a deployment-phase task.
