package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	configRegistryURL          = "registry.url"
	configRegistryNamespace    = "registry.namespace"
	configRegistryRepositories = "registry.repositories"
	configRegistryType         = "registry.type"
	configRegistryTLSVerify    = "registry.tlsVerify"
	configRegistrySyncInterval = "registry.syncInterval"
	configServerPort           = "server.port"
	configServerDBPath         = "server.dbPath"
)

var configEnvironment = map[string]string{
	configRegistryURL:          "SKILLCTL_REGISTRY_URL",
	configRegistryNamespace:    "SKILLCTL_REGISTRY_NAMESPACE",
	configRegistryRepositories: "SKILLCTL_REGISTRY_REPOSITORIES",
	configRegistryType:         "SKILLCTL_REGISTRY_TYPE",
	configRegistryTLSVerify:    "SKILLCTL_REGISTRY_TLS_VERIFY",
	configRegistrySyncInterval: "SKILLCTL_REGISTRY_SYNC_INTERVAL",
	configServerPort:           "SKILLCTL_SERVER_PORT",
	configServerDBPath:         "SKILLCTL_SERVER_DB_PATH",
}

func newConfig() *viper.Viper {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	if configDir, err := os.UserConfigDir(); err == nil {
		v.AddConfigPath(filepath.Join(configDir, "skillctl"))
	}

	for key, environment := range configEnvironment {
		if err := v.BindEnv(key, environment); err != nil {
			panic(fmt.Sprintf("bind environment variable %s: %v", environment, err))
		}
	}

	return v
}

func loadConfig(v *viper.Viper, configFile string) error {
	if configFile != "" {
		v.SetConfigFile(configFile)
	}

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if configFile == "" && errors.As(err, &notFound) {
			return nil
		}
		return fmt.Errorf("read config: %w", err)
	}

	return nil
}

func bindConfigFlag(v *viper.Viper, key string, flag *pflag.Flag) {
	if err := v.BindPFlag(key, flag); err != nil {
		panic(fmt.Sprintf("bind flag --%s: %v", flag.Name, err))
	}
}
