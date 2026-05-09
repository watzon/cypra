# Railway One-Click Template

The Railway template repository is published at [`watzon/cypra-railway-template`](https://github.com/watzon/cypra-railway-template).

The template repository includes:

- `railway.json` with Cypra and Postgres services.
- `STORAGE_BACKEND=s3-compatible` by default.
- R2/B2 setup instructions for bucket, endpoint, region, access key, and secret key.
- A README that walks bootstrap token redemption, tenant creation, provider configuration, and Next.js OIDC setup.

Live Railway beta deploy verification is still required before external tester rollout. It was deferred with owner approval during Phase R0 because the Railway CLI was not authenticated in the local agent environment.
