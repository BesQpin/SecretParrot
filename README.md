# Secret Parrot

<img src="SecretParrot.png" alt="alt text" style="width:25%;"/>

**Secret Parrot** is a Go application that copies secrets from one Azure Key Vault to multiple target Key Vaults.  
It supports managed identity authentication and is designed to run in Kubernetes or any containerized environment.

## Features

- Copy secrets from a source Key Vault to multiple targets
- Optionally copy only the latest version of each secret
- Include/exclude specific secrets via glob patterns
- Dry-run mode for testing
- Concurrent copying with configurable concurrency
- Support for managed identities

## Installation

### Build locally

```bash
git clone https://github.com/your-org/secret-parrot.git
cd secret-parrot
go build -o secret-parrot ./cmd/secret-parrot
```

### Build Docker image

```bash
docker build -t secret-parrot:latest .
```

## Usage

```bash
./secret-parrot \
  --source-vault <source-vault-name> \
  --target-vaults <comma-separated-target-vaults> \
  [--include <glob>] \
  [--exclude <glob>] \
  [--latest-only] \
  [--dry-run] \
  [--concurrency <number>]
```

### Flags

| Flag | Description |
|------|-------------|
| `--source-vault` | Name of the source Azure Key Vault |
| `--target-vaults` | Comma-separated list of target Azure Key Vaults |
| `--include` | Glob pattern of secrets to include |
| `--exclude` | Glob pattern of secrets to exclude |
| `--latest-only` | Copy only the latest version of each secret |
| `--dry-run` | Print actions without modifying target Key Vaults |
| `--concurrency` | Number of concurrent secret copy operations (default: 8) |

## Kubernetes Deployment

You can containerize the app and run it in a Kubernetes cluster. Example manifest:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: secret-parrot
spec:
  replicas: 1
  selector:
    matchLabels:
      app: secret-parrot
  template:
    metadata:
      labels:
        app: secret-parrot
    spec:
      containers:
      - name: secret-parrot
        image: secret-parrot:latest
        args:
        - "--source-vault"
        - "source-vault-name"
        - "--target-vaults"
        - "target1,target2"
        env:
        - name: AZURE_CLIENT_ID
          valueFrom:
            secretKeyRef:
              name: azure-credentials
              key: clientId
        - name: AZURE_TENANT_ID
          valueFrom:
            secretKeyRef:
              name: azure-credentials
              key: tenantId
        - name: AZURE_CLIENT_SECRET
          valueFrom:
            secretKeyRef:
              name: azure-credentials
              key: clientSecret
```

## License

MIT License