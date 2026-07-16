package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
)

func TestLoadConfig(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "skillctl.yaml")
	contents := []byte(`registry:
  url: quay.io/example
  namespace: team
  tlsVerify: false
  syncInterval: 5m
server:
  port: 9090
  dbPath: /tmp/catalog.db
`)
	if err := os.WriteFile(configFile, contents, 0o600); err != nil {
		t.Fatal(err)
	}

	v := newConfig()
	if err := loadConfig(v, configFile); err != nil {
		t.Fatal(err)
	}

	checks := map[string]any{
		configRegistryURL:          "quay.io/example",
		configRegistryNamespace:    "team",
		configRegistryTLSVerify:    false,
		configRegistrySyncInterval: "5m",
		configServerPort:           9090,
		configServerDBPath:         "/tmp/catalog.db",
	}
	for key, want := range checks {
		if got := v.Get(key); got != want {
			t.Errorf("%s = %#v, want %#v", key, got, want)
		}
	}
}

func TestConfigPrecedence(t *testing.T) {
	t.Setenv("SKILLCTL_SERVER_PORT", "9090")
	configFile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configFile, []byte("server:\n  port: 7070\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	v := newConfig()
	if err := loadConfig(v, configFile); err != nil {
		t.Fatal(err)
	}

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.Int("port", 8080, "")
	bindConfigFlag(v, configServerPort, flags.Lookup("port"))

	if got := v.GetInt(configServerPort); got != 9090 {
		t.Fatalf("environment value = %d, want 9090", got)
	}

	if err := flags.Set("port", "6060"); err != nil {
		t.Fatal(err)
	}
	if got := v.GetInt(configServerPort); got != 6060 {
		t.Fatalf("flag value = %d, want 6060", got)
	}
}

func TestLoadConfigMissingExplicitFile(t *testing.T) {
	v := newConfig()
	if err := loadConfig(v, filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatal("expected an error for a missing explicit config file")
	}
}

func TestRootCommandUsesCobraAndExposesConfigFlag(t *testing.T) {
	cmd := NewRootCmd("test")
	if cmd.PersistentFlags().Lookup("config") == nil {
		t.Fatal("root command is missing --config")
	}
	if cmd.Commands() == nil {
		t.Fatal("root command has no Cobra subcommands")
	}
}
