# Secret Parrot

<img src="SecretParrot.png" alt="alt text" style="width:25%;"/>

**Secret Parrot** is a Go application that copies secrets from one Azure Key Vault to multiple target Key Vaults.  
It supports multiple authentication methods including managed identity, Azure CLI credentials, and interactive browser login.

## Features

- Copy secrets from a source Key Vault to multiple targets
- Optionally copy only the latest version of each secret
- Include/exclude specific secrets via glob patterns
- Dry-run mode for testing
- Concurrent copying with configurable concurrency
- Flexible authentication options:
  - Managed identity (for containerized environments)
  - Environment credentials
  - Azure CLI credentials (for local development)
  - Interactive browser login (fallback)

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

### Local Development

When running locally, Secret Parrot will automatically use your Azure CLI credentials if you're already logged in:

```bash
# Using existing Azure CLI credentials
az login  # if not already logged in
./secret-parrot \
  --source-vault <source-vault-name> \
  --target-vaults <comma-separated-target-vaults>
```

If no Azure CLI credentials are found, it will fall back to interactive browser login unless disabled:

```bash
# Disable browser-based authentication
NO_BROWSER_AUTH=true ./secret-parrot \
  --source-vault <source-vault-name> \
  --target-vaults <comma-separated-target-vaults>
```

### Container Environment

When running in a container, use environment variables for authentication:

```bash
docker run secret-parrot:latest \
  -e AZURE_TENANT_ID=<tenant-id> \
  -e AZURE_CLIENT_ID=<client-id> \
  -e AZURE_CLIENT_SECRET=<client-secret> \
  --source-vault <source-vault-name> \
  --target-vaults <target1,target2>
```

### Common Flags

| Flag | Description |
|------|-------------|
| `-source`             | Name of the source Azure Key Vault |
| `-targets`            | Comma-separated list of target Azure Key Vaults |
| `-include`            | Glob pattern of secrets to include |
| `-exclude`            | Glob pattern of secrets to exclude |
| `-latest-only`        | Copy only the latest version of each secret |
| `-override-disabled`  | Copy even if source secret is disabled |
| `-dry-run`            | Print actions without modifying target Key Vaults |
| `-concurrency`        | Number of concurrent secret copy operations (default: 8) |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `AZURE_TENANT_ID` | Azure tenant ID (optional for CLI auth) |
| `AZURE_CLIENT_ID` | Service principal client ID |
| `AZURE_CLIENT_SECRET` | Service principal client secret |
| `NO_BROWSER_AUTH` | Set to "true" to disable interactive browser login |
| `DEBUG` | Set to "true" for debug logging |

## Authentication Flow

Secret Parrot attempts authentication methods in the following order:

1. Environment credentials (using `AZURE_*` environment variables)
2. Azure CLI credentials (if running locally and logged in)
3. Interactive browser login (unless disabled with `NO_BROWSER_AUTH=true`)

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
        - name: NO_BROWSER_AUTH
          value: "true"
```

## License

MIT License