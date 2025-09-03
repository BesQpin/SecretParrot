package copier

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/your-org/secret-parrot/internal/kv"
)

type Logger interface {
	Printf(string, ...any)
	Fatalf(string, ...any)
}

type Copier struct {
	Credential       azsecrets.TokenCredential
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

	pager := srcClient.NewListSecretsPager(nil)
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
			name := *item.ID.Name
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
	ver, err := src.GetSecret(ctx, name, "", nil) // latest
	if err != nil {
		recordErr(mu, firstErr, fmt.Errorf("get %s: %w", name, err))
		return
	}

	if ver.Properties.Enabled != nil && !*ver.Properties.Enabled && !c.OverrideDisabled {
		return
	}

	for tName, tClient := range targets {
		if c.DryRun {
			c.Logger.Printf("DRY-RUN copy %s -> %s", name, tName)
			continue
		}
		if err := kv.CopySecret(ctx, tClient, name, *ver.Value, ver.Properties.ContentType, ver.Properties.Enabled, toKVTags(ver.Properties.Tags)); err != nil {
			recordErr(mu, firstErr, fmt.Errorf("set %s in %s: %w", name, tName, err))
		}
	}
}

func copyAllVersions(ctx context.Context, c *Copier, src *azsecrets.Client, targets map[string]*azsecrets.Client, name string, mu *sync.Mutex, firstErr *error) {
	pager := src.NewListSecretVersionsPager(name, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			recordErr(mu, firstErr, fmt.Errorf("list versions %s: %w", name, err))
			return
		}
		for _, v := range page.Value {
			ver, err := src.GetSecret(ctx, name, *v.ID.Version, nil)
			if err != nil {
				recordErr(mu, firstErr, fmt.Errorf("get %s/%s: %w", name, *v.ID.Version, err))
				continue
			}
			if ver.Properties.Enabled != nil && !*ver.Properties.Enabled && !c.OverrideDisabled {
				continue
			}
			for tName, tClient := range targets {
				if c.DryRun {
					c.Logger.Printf("DRY-RUN copy %s@%s -> %s", name, *v.ID.Version, tName)
					continue
				}
				if err := kv.CopySecret(ctx, tClient, name, *ver.Value, ver.Properties.ContentType, ver.Properties.Enabled, toKVTags(ver.Properties.Tags)); err != nil {
					recordErr(mu, firstErr, fmt.Errorf("set %s@%s in %s: %w", name, *v.ID.Version, tName, err))
				}
			}
		}
	}
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
	for _, p := range filepath.SplitList(strings.ReplaceAll(s, ",", string(filepath.ListSeparator))) {
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
	ok, err := filepath.Match(glob, s)
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
