package config

import (
	"reflect"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	// "go.temporal.io/sdk/client"
)

type (
	Provider interface {
		Config() Config
		Logger() zerolog.Logger
	}

	Config struct {
		Server ServerConfig `mapstructure:"server"`

		Persistence PersistenceConfig `mapstructure:"persistence"`
		// Temporal    TemporalConfig    `mapstructure:"temporal"`
		// Kafka       KafkaConfig       `mapstructure:"kafka"`
	}

	ServerConfig struct {
		HTTPAddress string `mapstructure:"httpAddress"`
	}

	PersistenceConfig struct {
		Postgres struct {
			DSN string `mapstructure:"dsn"`
		} `mapstructure:"postgres"`
	}

	//
	// TemporalConfig struct {
	// 	Disabled        bool   `mapstructure:"disabled"`
	// 	HostPort        string `mapstructure:"hostPort"`
	// 	Namespace       string `mapstructure:"namespace"`
	// 	WorkerTaskQueue string `mapstructure:"workerTaskQueue"`
	// }
	//
	// KafkaConfig struct {
	// 	Disabled bool           `mapstructure:"disabled"`
	// 	Brokers  []string       `mapstructure:"brokers"`
	// 	GroupID  string         `mapstructure:"groupId"`
	// 	SASL     *KafkaSASLAuth `mapstructure:"sasl"`
	// }
	//
	// KafkaSASLAuth struct {
	// 	Mechanism string `mapstructure:"mechanism"`
	// 	Username  string `mapstructure:"username" sensitive:"true"`
	// 	Password  string `mapstructure:"password" sensitive:"true"`
	// }
)

func New() (Config, error) {
	config := new(Config)
	if err := viper.Unmarshal(config); err != nil {
		return *config, err
	}
	return *config, nil
}

func (c Config) LogValue() map[string]any {
	return maskSensitive(reflect.ValueOf(c), reflect.TypeOf(c))
}

func maskSensitive(v reflect.Value, t reflect.Type) map[string]any {
	result := make(map[string]any, t.NumField())
	for i := range t.NumField() {
		field := t.Field(i)
		val := v.Field(i)

		key := field.Tag.Get("mapstructure")
		if key == "" {
			key = field.Name
		}

		if field.Tag.Get("sensitive") == "true" {
			result[key] = "***"
			continue
		}

		switch val.Kind() {
		case reflect.Struct:
			result[key] = maskSensitive(val, field.Type)
		case reflect.Pointer:
			if val.IsNil() {
				result[key] = nil
			} else {
				result[key] = maskSensitive(val.Elem(), field.Type.Elem())
			}
		default:
			result[key] = val.Interface()
		}
	}
	return result
}

//
// func (c *Config) TemporalClientOptions() client.Options {
// 	return client.Options{
// 		HostPort:  c.Temporal.HostPort,
// 		Namespace: c.Temporal.Namespace,
// 	}
// }
