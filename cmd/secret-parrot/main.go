package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/your-org/secret-parrot/internal/auth"
	"github.com/your-org/secret-parrot/internal/config"
	"github.com/your-org/secret-parrot/internal/copier"
	"github.com/your-org/secret-parrot/internal/logging"
)

func main() {
	cfg := config.FromEnv()

	// CLI flags override env
	var (
		source   = flag.String("source", cfg.SourceVault, "Source Key Vault name (without https:// and domain)")
		targets  = flag.String("targets", strings.Join(cfg.TargetVaults, ","), "Comma-separated target Key Vault names")
		include  = flag.String("include", cfg.Include, "Comma-separated glob patterns to include (e.g. 'app-*')")
		exclude  = flag.String("exclude", cfg.Exclude, "Comma-separated glob patterns to exclude")
		dryRun   = flag.Bool("dry-run", cfg.DryRun, "If true, do not write to targets")
		concur   = flag.Int("concurrency", cfg.Concurrency, "Max concurrent operations")
		override = flag.Bool("override-disabled", cfg.OverrideDisabled, "Copy even if source secret is disabled")
		latest   = flag.Bool("latest-only", cfg.LatestOnly, "Copy only latest versions (true) or all versions (false)")
	)
	flag.Parse()

	if *source == "" {
		log.Fatal("--source is required (or SOURCE_VAULT env var)")
	}
	var targetList []string
	if *targets != "" {
		for _, t := range strings.Split(*targets, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				targetList = append(targetList, t)
			}
		}
	}
	if len(targetList) == 0 {
		log.Fatal("--targets is required (comma-separated) or TARGET_VAULTS env var")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	logger := logging.New()
	cred, err := auth.NewCredential()
	if err != nil {
		logger.Fatalf("auth: %v", err)
	}

	c := copier.Copier{
		Credential:       cred,
		SourceVaultName:  *source,
		TargetVaultNames: targetList,
		Include:          *include,
		Exclude:          *exclude,
		DryRun:           *dryRun,
		Concurrency:      *concur,
		OverrideDisabled: *override,
		LatestOnly:       *latest,
		Logger:           logger,
	}

	if err := c.Run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "secret-parrot error:", err)
		os.Exit(1)
	}
}
