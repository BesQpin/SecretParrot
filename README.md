# Secret Parrot

Copy secrets from one Key Vault to many.

## Build

```bash
go build ./cmd/secret-parrot
````

## Container

```bash
docker build -t secret-parrot:local .
```

## Run locally (uses DefaultAzureCredential)

```bash
export SOURCE_VAULT=src-vault
export TARGET_VAULTS=dst-a,dst-b
./secret-parrot --include="app-*" --concurrency=16 --latest-only=true
```

Common flags/env:

* `SOURCE_VAULT` / `--source` – source Key Vault name
* `TARGET_VAULTS` / `--targets` – comma-separated target Key Vault names
* `INCLUDE_PATTERNS` / `--include` – globs to select which secret names to copy
* `EXCLUDE_PATTERNS` / `--exclude` – globs to skip secrets
* `DRY_RUN` / `--dry-run` – don’t write, just log
* `CONCURRENCY` / `--concurrency` – parallel operations (default 8)
* `LATEST_ONLY` / `--latest-only` – copy only current version (default true)
* `OVERRIDE_DISABLED` / `--override-disabled` – copy even if source secret is disabled

### Notes

* Secrets’ **content type** and **tags** are preserved. Enabled/disabled state is preserved unless `--override-disabled=false` prevents it.
* By default only the latest version is copied; use `--latest-only=false` to migrate **all versions**.
* On targets, existing secrets will be **updated** (new version created) with the source value.
* If a secret exists in targets but not in source (e.g., drift), Secret Parrot does not delete it.
* Retries/backoffs rely on Azure SDK defaults; you can wrap with a controller or CronJob in K8s.

## Kubernetes (Workload Identity)

* Create a federated identity credential mapping your cluster’s issuer and the `ServiceAccount` to your managed identity.
* Assign Key Vault roles to the managed identity as above.
* Apply `k8s/secret-parrot-config.example.yaml` and `k8s/deployment.yaml`.

````

---

## Future enhancements (backlog)

- Metrics endpoint (Prometheus) with copy counts and durations.
- Exponential backoff/retry with per-secret error classification.
- Partial sync (only tags/content-type update) and delete-orphans mode (opt-in).
- Expose as HTTP service to trigger sync via webhook in addition to CLI run.
- Per-vault concurrency limits to avoid throttling (429) on large fan-out.
- Optional AES encryption at rest for transit artifacts (if ever added).
```

````