package copier

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/your-org/secret-parrot/internal/kv"
)

type Logger interface {
	Printf(string, ...any)
	Fatalf(string, ...any)
}

type Copier struct {
	Credential       azcore.TokenCredential
	SourceVaultName  string
	TargetVaultNames []string
	Include          string
	Exclude          string
	DryRun           bool
	Concurrency      int
	OverrideDisabled bool
	LatestOnly       bool
	Logger           Logger
}

func (c *Copier) Run(ctx context.Context) error {
	if c.SourceVaultName == "" || len(c.TargetVaultNames) == 0 {
		return errors.New("source and targets required")
	}
	if c.Concurrency <= 0 {
		c.Concurrency = 8
	}

	srcClient, err := azsecrets.NewClient(kv.VaultURL(c.SourceVaultName), c.Credential, nil)
	if err != nil {
		return fmt.Errorf("source client: %w", err)
	}

	tgtClients := make(map[string]*azsecrets.Client, len(c.TargetVaultNames))
	for _, name := range c.TargetVaultNames {
		cl, err := azsecrets.NewClient(kv.VaultURL(name), c.Credential, nil)
		if err != nil {
			return fmt.Errorf("target client %s: %w", name, err)
		}
		tgtClients[name] = cl
	}

	inc := splitList(c.Include)
	exc := splitList(c.Exclude)

	pager := srcClient.NewListSecretPropertiesPager(nil)
	sem := make(chan struct{}, c.Concurrency)
	var wg sync.WaitGroup
	var firstErr error
	var mu sync.Mutex

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("list secrets: %w", err)
		}
		for _, item := range page.Value {
			if item.ID == nil {
				continue
			}
			id := string(*item.ID)
			name, err := extractName(id)
			if err != nil {
				recordErr(&mu, &firstErr, fmt.Errorf("parse id: %w", err))
				continue
			}
			if !allow(name, inc, exc) {
				continue
			}

			wg.Add(1)
			sem <- struct{}{}
			go func(name string) {
				defer wg.Done()
				defer func() { <-sem }()
				if c.LatestOnly {
					copyLatest(ctx, c, srcClient, tgtClients, name, &mu, &firstErr)
					return
				}
				copyAllVersions(ctx, c, srcClient, tgtClients, name, &mu, &firstErr)
			}(name)
		}
	}
	wg.Wait()
	return firstErr
}

func copyLatest(ctx context.Context, c *Copier, src *azsecrets.Client, targets map[string]*azsecrets.Client, name string, mu *sync.Mutex, firstErr *error) {
	ver, err := src.GetSecret(ctx, name, "", nil)
	if err != nil {
		recordErr(mu, firstErr, fmt.Errorf("get %s: %w", name, err))
		return
	}

	if ver.Attributes != nil && ver.Attributes.Enabled != nil && !*ver.Attributes.Enabled && !c.OverrideDisabled {
		return
	}

	for tName, tClient := range targets {
		if c.DryRun {
			c.Logger.Printf("DRY-RUN copy %s -> %s", name, tName)
			continue
		}
		tags := toKVTags(ver.Tags)
		if err := kv.CopySecret(ctx, tClient, name, *ver.Value, ver.ContentType, getEnabledFromAttributes(ver.Attributes), tags); err != nil {
			recordErr(mu, firstErr, fmt.Errorf("set %s in %s: %w", name, tName, err))
		}
	}
}

func copyAllVersions(ctx context.Context, c *Copier, srcClient *azsecrets.Client, targets map[string]*azsecrets.Client, name string, mu *sync.Mutex, firstErr *error) {
	pager := srcClient.NewListSecretPropertiesVersionsPager(name, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			recordErr(mu, firstErr, fmt.Errorf("list versions %s: %w", name, err))
			return
		}
		for _, v := range page.Value {
			if v.ID == nil {
				continue
			}
			versionID := string(*v.ID)
			version, err := extractVersion(versionID)
			if err != nil {
				recordErr(mu, firstErr, fmt.Errorf("parse version: %w", err))
				continue
			}

			ver, err := srcClient.GetSecret(ctx, name, version, nil)
			if err != nil {
				recordErr(mu, firstErr, fmt.Errorf("get %s/%s: %w", name, version, err))
				continue
			}
			if ver.Attributes != nil && ver.Attributes.Enabled != nil && !*ver.Attributes.Enabled && !c.OverrideDisabled {
				continue
			}
			for tName, tClient := range targets {
				if c.DryRun {
					c.Logger.Printf("DRY-RUN copy %s@%s -> %s", name, version, tName)
					continue
				}
				tags := toKVTags(ver.Tags)
				if err := kv.CopySecret(ctx, tClient, name, *ver.Value, ver.ContentType, getEnabledFromAttributes(ver.Attributes), tags); err != nil {
					recordErr(mu, firstErr, fmt.Errorf("set %s@%s in %s: %w", name, version, tName, err))
				}
			}
		}
	}
}

func getEnabledFromAttributes(attrs *azsecrets.SecretAttributes) *bool {
	if attrs == nil {
		return nil
	}
	return attrs.Enabled
}

func extractName(id string) (string, error) {
	u, err := url.Parse(id)
	if err != nil {
		return "", err
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || parts[0] != "secrets" {
		return "", fmt.Errorf("unexpected secret ID format: %s", id)
	}
	return parts[1], nil
}

func extractVersion(id string) (string, error) {
	u, err := url.Parse(id)
	if err != nil {
		return "", err
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "secrets" {
		return "", fmt.Errorf("unexpected secret version ID format: %s", id)
	}
	return parts[2], nil
}

func recordErr(mu *sync.Mutex, firstErr *error, err error) {
	mu.Lock()
	defer mu.Unlock()
	if *firstErr == nil {
		*firstErr = err
	}
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func allow(name string, include, exclude []string) bool {
	if len(include) > 0 {
		ok := false
		for _, g := range include {
			if match(g, name) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	for _, g := range exclude {
		if match(g, name) {
			return false
		}
	}
	return true
}

func match(glob, s string) bool {
	ok, err := path.Match(glob, s)
	return err == nil && ok
}

func toKVTags(in map[string]*string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		if v != nil {
			out[k] = *v
		}
	}
	return out
}
