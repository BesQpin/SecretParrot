package kv

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
)

func VaultURL(name string) string {
	name = strings.TrimSpace(name)
	return fmt.Sprintf("https://%s.vault.azure.net/", name)
}

func NewSecretsClient(vaultName string, cred azsecrets.TokenCredential) (*azsecrets.Client, error) {
	return azsecrets.NewClient(VaultURL(vaultName), cred, nil)
}

// CopySecret sets a secret on target vault with properties/tags from source.
func CopySecret(ctx context.Context, target *azsecrets.Client, name string, value string, contentType *string, enabled *bool, tags map[string]string) error {
	var kvTags map[string]*string
	if len(tags) > 0 {
		kvTags = map[string]*string{}
		for k, v := range tags {
			vv := v
			kvTags[k] = &vv
		}
	}
	_, err := target.SetSecret(ctx, name, azsecrets.SetSecretParameters{
		Value:       to.Ptr(value),
		ContentType: contentType,
		Enabled:     enabled,
		Tags:        kvTags,
	}, nil)
	return err
}
