package auth

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// NewCredential returns a DefaultAzureCredential that works with
// managed identity (in AKS/VM), Azure CLI, Visual Studio Code, etc.
func NewCredential() (*azidentity.DefaultAzureCredential, error) {
	return azidentity.NewDefaultAzureCredential(nil)
}
