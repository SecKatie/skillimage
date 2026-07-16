package cli

import (
	"context"
	"fmt"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/redhat-et/skillimage/internal/server"
	"github.com/redhat-et/skillimage/pkg/oci"
)

func newServeCmd(v *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the skill catalog server",
		Long: `Start an HTTP server that indexes skills from an OCI registry
and serves them via a REST API.

The server syncs skill metadata from the configured registry into
a local SQLite database and serves it for fast listing, filtering,
and search.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			syncInterval := v.GetString(configRegistrySyncInterval)
			interval, err := time.ParseDuration(syncInterval)
			if err != nil {
				return fmt.Errorf("invalid sync interval: %w", err)
			}

			registryURL := v.GetString(configRegistryURL)
			if registryURL == "" {
				return fmt.Errorf("--registry is required")
			}

			ctx, cancel := signal.NotifyContext(
				context.Background(), syscall.SIGINT, syscall.SIGTERM,
			)
			defer cancel()

			registryType := v.GetString(configRegistryType)
			rt := oci.RegistryType(registryType)
			switch rt {
			case oci.RegistryTypeAuto, oci.RegistryTypeOCI, oci.RegistryTypeQuay:
			default:
				return fmt.Errorf("invalid --registry-type %q, must be auto, oci, or quay", registryType)
			}

			var repos []string
			if repositories := v.GetString(configRegistryRepositories); repositories != "" {
				for _, r := range strings.Split(repositories, ",") {
					if s := strings.TrimSpace(r); s != "" {
						repos = append(repos, s)
					}
				}
			}

			return server.Run(ctx, server.Config{
				Port:          v.GetInt(configServerPort),
				DBPath:        v.GetString(configServerDBPath),
				RegistryURL:   registryURL,
				Namespace:     v.GetString(configRegistryNamespace),
				Repositories:  repos,
				SkipTLSVerify: !v.GetBool(configRegistryTLSVerify),
				SyncInterval:  interval,
				RegistryType:  rt,
			})
		},
	}

	flags := cmd.Flags()
	flags.Int("port", 8080, "HTTP listen port")
	flags.String("db", "skillctl.db", "SQLite database path")
	flags.String("registry", "", "OCI registry URL (required)")
	flags.String("namespace", "", "namespace for discovery (Quay: org name; OCI: prefix filter for /v2/_catalog)")
	flags.String("repositories", "", "comma-separated repo names (bypasses discovery)")
	flags.String("registry-type", "auto", `registry discovery type: "auto", "oci", or "quay"`)
	flags.String("sync-interval", "60s", "background sync interval")
	flags.Bool("tls-verify", true, "require HTTPS and verify certificates")

	bindConfigFlag(v, configServerPort, flags.Lookup("port"))
	bindConfigFlag(v, configServerDBPath, flags.Lookup("db"))
	bindConfigFlag(v, configRegistryURL, flags.Lookup("registry"))
	bindConfigFlag(v, configRegistryNamespace, flags.Lookup("namespace"))
	bindConfigFlag(v, configRegistryRepositories, flags.Lookup("repositories"))
	bindConfigFlag(v, configRegistryType, flags.Lookup("registry-type"))
	bindConfigFlag(v, configRegistrySyncInterval, flags.Lookup("sync-interval"))
	bindConfigFlag(v, configRegistryTLSVerify, flags.Lookup("tls-verify"))

	return cmd
}
