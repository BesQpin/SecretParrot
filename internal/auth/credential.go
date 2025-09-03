package auth

import (
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// NewCredential creates a new TokenCredential using the following precedence:
// 1. Environment variables (client ID, client secret, tenant ID)
// 2. Azure CLI credentials
// 3. Interactive browser login
func NewCredential() (*azidentity.ChainedTokenCredential, error) {
	// Create credential chain options
	var creds []azcore.TokenCredential

	// Try environment credentials first (for container scenarios)
	if envCred, err := azidentity.NewEnvironmentCredential(nil); err == nil {
		creds = append(creds, envCred)
	}

	// Try Azure CLI credentials (for development)
	if cliCred, err := azidentity.NewAzureCLICredential(nil); err == nil {
		creds = append(creds, cliCred)
	}

	// Add interactive browser login as fallback
	if _, ok := os.LookupEnv("NO_BROWSER_AUTH"); !ok {
		if interactiveCred, err := azidentity.NewInteractiveBrowserCredential(&azidentity.InteractiveBrowserCredentialOptions{
			TenantID: os.Getenv("AZURE_TENANT_ID"), // Optional, will use default tenant if not set
		}); err == nil {
			creds = append(creds, interactiveCred)
		}
	}

	if len(creds) == 0 {
		return nil, fmt.Errorf("no valid authentication methods available")
	}

	// Create chained credential
	chain, err := azidentity.NewChainedTokenCredential(creds, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create chained credential: %w", err)
	}

	return chain, nil
}
