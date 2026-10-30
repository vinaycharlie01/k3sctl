package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/vinaycharlie01/k3sctl/pkg/apis/k3sctl.k3sproject.io/v1beta1"
	"github.com/vinaycharlie01/k3sctl/pkg/manifest"
)

func GlobalFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    "debug",
			Usage:   "Enable debug logging",
			EnvVars: []string{"K3SCTL_DEBUG"},
		},
		&cli.BoolFlag{
			Name:    "trace",
			Usage:   "Enable trace logging",
			EnvVars: []string{"K3SCTL_TRACE"},
		},
		&cli.BoolFlag{
			Name:  "no-redact",
			Usage: "Disable log redaction",
		},
	}
}

func configFlag() *cli.StringFlag {
	return &cli.StringFlag{
		Name:      "config",
		Aliases:   []string{"c"},
		Usage:     "Path to cluster configuration `FILE`",
		Value:     "k3sctl.yaml",
		TakesFile: true,
		EnvVars:   []string{"K3SCTL_CONFIG"},
	}
}

func concurrencyFlag() *cli.IntFlag {
	return &cli.IntFlag{
		Name:  "concurrency",
		Usage: "Maximum number of parallel hosts to process",
		Value: 30,
	}
}

func concurrentUploadsFlag() *cli.IntFlag {
	return &cli.IntFlag{
		Name:  "concurrent-uploads",
		Usage: "Maximum number of concurrent file uploads",
		Value: 5,
	}
}

func dryRunFlag() *cli.BoolFlag {
	return &cli.BoolFlag{
		Name:  "dry-run",
		Usage: "Print what would be done without making changes",
	}
}

func initLogging(ctx *cli.Context) error {
	level := slog.LevelInfo
	if ctx.Bool("trace") || ctx.Bool("debug") {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))

	return nil
}

func loadConfig(configPath string) (*v1beta1.Cluster, error) {
	paths, err := filepath.Glob(configPath)
	if err != nil {
		return nil, fmt.Errorf("glob config path: %w", err)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	reader := &manifest.Reader{}
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			return nil, fmt.Errorf("open config %s: %w", p, err)
		}
		defer func() { _ = f.Close() }()
		if err := reader.Parse(f); err != nil {
			return nil, fmt.Errorf("parse config %s: %w", p, err)
		}
	}

	resources := reader.Resources()

	var clusterResources []*manifest.ResourceDefinition
	for _, rd := range resources {
		if strings.EqualFold(rd.APIVersion, v1beta1.APIVersion) && strings.EqualFold(rd.Kind, "Cluster") {
			clusterResources = append(clusterResources, rd)
		}
	}
	if len(clusterResources) != 1 {
		return nil, fmt.Errorf("expected exactly one %s Cluster resource, got %d", v1beta1.APIVersion, len(clusterResources))
	}

	cfg := &v1beta1.Cluster{}
	if err := clusterResources[0].Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	for _, rd := range resources {
		if rd == clusterResources[0] {
			continue
		}
		if cfg.Metadata.Manifests == nil {
			cfg.Metadata.Manifests = make(map[string][]byte)
		}
		name := rd.Filename()
		for i := 2; ; i++ {
			if _, exists := cfg.Metadata.Manifests[name]; !exists {
				break
			}
			name = fmt.Sprintf("%s-%d", rd.Filename(), i)
		}
		cfg.Metadata.Manifests[name] = rd.Raw
	}

	cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}

func displayLogo() {
	logo := `
 _    _____          _   _
| | _|___ / ___  ___| |_| |
| |/ / |_ \/ __|/ __| __| |
|   < ___) \__ \ (__| |_| |
|_|\_\____/|___/\___|\__|_|
`
	fmt.Fprintln(os.Stderr, strings.TrimRight(logo, "\n"))
}
