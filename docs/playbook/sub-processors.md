# Sub-Processors

Copy this template into your deployment docs and fill it with the providers you configure. The operator is responsible for the DPA, region, retention, and support contact for each provider.

| Provider       | Purpose          | Data categories                                | Region          | Contract / DPA | Notes                                      |
| -------------- | ---------------- | ---------------------------------------------- | --------------- | -------------- | ------------------------------------------ |
| Resend         | Email delivery   | Email address, template payload, delivery logs | Operator-chosen | TBD            | Required only when Resend email is enabled |
| Object storage | Profile pictures | Binary objects, object metadata                | Operator-chosen | TBD            | Required only for S3-compatible storage    |
| Log platform   | Log retention    | Request IDs, audit context, redacted tokens    | Operator-chosen | TBD            | Redact `redacted-on-export` values         |
| OTEL collector | Tracing          | Request spans, route names, tenant context     | Operator-chosen | TBD            | Optional                                   |

Review this table before each external tester wave and after enabling a new provider.
