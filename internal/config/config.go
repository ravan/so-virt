package config

import (
	"github.com/alecthomas/kingpin"
	"github.com/go-playground/validator/v10"
	"github.com/ravan/stackstate-client/stackstate"
	"github.com/ravan/stackstate-client/stackstate/receiver"
	"github.com/spf13/viper"
	"k8s.io/client-go/util/homedir"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Configuration struct {
	SuseObservability stackstate.StackState `mapstructure:"suseobservability" validate:"required"`
	Instance          receiver.Instance     `mapstructure:"instance" validate:"required"`
	Kubernetes        Kubernetes            `mapstructure:"kubernetes" validate:"required"`
}

type Kubernetes struct {
	Cluster    string `mapstructure:"cluster" validate:"required"`
	KubeConfig string `mapstructure:"kubeconfig"`
	InCluster  bool   `mapstructure:"in_cluster"`
}

func GetConfig() (*Configuration, error) {
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		cf := kingpin.Flag("config-file", "config file").Short('c').ExistingFile()
		if *cf != "" {
			configFile = *cf
		}
	}
	if configFile != "" {
		slog.Info("Using config file", "file", configFile)
	}

	c := &Configuration{Instance: receiver.Instance{}}
	v := viper.New()
	if home := homedir.HomeDir(); home != "" {
		v.SetDefault("kubernetes.kubeconfig", filepath.Join(home, ".kube", "config"))
	} else {
		v.SetDefault("kubernetes.kubeconfig", "")
	}
	v.SetDefault("kubernetes.in_cluster", false)
	v.SetDefault("kubernetes.cluster", "")
	v.SetDefault("suseobservability.api_url", "")
	v.SetDefault("suseobservability.api_key", "")
	v.SetDefault("suseobservability.api_token", "")
	v.SetDefault("instance.type", "virt")
	v.SetDefault("instance.url", "virt")

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	if configFile != "" {
		d, f := path.Split(configFile)
		if d == "" {
			d = "."
		}
		v.SetConfigName(f[0 : len(f)-len(filepath.Ext(f))])
		v.AddConfigPath(d)
		err := v.ReadInConfig()
		if err != nil {
			slog.Error("Error when reading config file.", slog.Any("error", err))
		}
	}

	if err := v.Unmarshal(c); err != nil {
		slog.Error("Error unmarshalling config", slog.Any("err", err))
		return nil, err
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	err := validate.Struct(c)
	if err != nil {
		return nil, err
	}
	return c, nil
}
