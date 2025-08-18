package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type APIConfig struct {
	ListenAddress string `mapstructure:"listen_address"`
	AuthToken     string `mapstructure:"auth_token"`
}

type StorageConfig struct {
	ImagesPath string `mapstructure:"images_path"`
	DisksPath  string `mapstructure:"disks_path"`
}

type NetworkConfig struct {
	Bridge string `mapstructure:"bridge"`
}

type LibvirtConfig struct {
	Address string `mapstructure:"address"`
}

type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

type GRPCConfig struct {
	ListenAddress string    `mapstructure:"listen_address"`
	TLS           TLSConfig `mapstructure:"tls"`
}

type Config struct {
	API     APIConfig     `mapstructure:"api"`
	Storage StorageConfig `mapstructure:"storage"`
	Network NetworkConfig `mapstructure:"network"`
	Libvirt LibvirtConfig `mapstructure:"libvirt"`
	GRPC    GRPCConfig    `mapstructure:"grpc"`
}

func defaultConfig() Config {
	return Config{
		API: APIConfig{
			ListenAddress: ":8080",
		},
		GRPC: GRPCConfig{
			ListenAddress: ":9090",
		},
		Storage: StorageConfig{
			ImagesPath: "/var/lib/deusvm/images",
			DisksPath:  "/var/lib/deusvm/disks",
		},
		Network: NetworkConfig{Bridge: "br0"},
	}
}

func Load() (Config, error) {
	cfg := defaultConfig()

	v := viper.New()
	v.SetConfigName("deusvm")
	v.AddConfigPath("/etc/deusvm/")
	v.AddConfigPath(".")
	v.SetConfigType("yaml")

	// env overrides like DEUSVM_API_LISTEN_ADDRESS, DEUSVM_STORAGE_IMAGES_PATH, etc
	v.SetEnvPrefix("DEUSVM")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		// ignore missing file; use defaults and env
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok && !os.IsNotExist(err) {
			return cfg, fmt.Errorf("read config: %w", err)
		}
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("unmarshal config: %w", err)
	}

	return cfg, nil
}
