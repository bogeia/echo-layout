package config

import (
	"fmt"
	"log/slog"
	"net"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

type Config struct {
	Server Server `mapstructure:"server"`
	Log    Log    `mapstructure:"log"`
}

type Server struct {
	Address         string        `mapstructure:"address"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

type Log struct {
	Level      slog.Level `mapstructure:"level"`
	Console    bool       `mapstructure:"console"`
	Filename   string     `mapstructure:"filename"`
	MaxSize    int        `mapstructure:"max_size"`
	MaxBackups int        `mapstructure:"max_backups"`
	MaxAge     int        `mapstructure:"max_age"`
	Compress   bool       `mapstructure:"compress"`
}

func Load(path string) (Config, error) {
	var config Config
	if ext := strings.ToLower(filepath.Ext(path)); ext != ".yaml" && ext != ".yml" {
		return config, fmt.Errorf("configuration must be a YAML file")
	}
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		return config, fmt.Errorf("read configuration: %w", err)
	}
	err := v.Unmarshal(&config, func(d *mapstructure.DecoderConfig) {
		d.ErrorUnset = true
		d.WeaklyTypedInput = false
		d.DecodeHook = func(from, to reflect.Type, value any) (any, error) {
			if to == reflect.TypeFor[slog.Level]() {
				if from.Kind() != reflect.String {
					return nil, fmt.Errorf("log level must be a string")
				}
				var level slog.Level
				err := level.UnmarshalText([]byte(value.(string)))
				return level, err
			}
			if to == reflect.TypeFor[time.Duration]() {
				if from.Kind() != reflect.String {
					return nil, fmt.Errorf("duration must be a string with units, such as 10s")
				}
				return time.ParseDuration(value.(string))
			}
			if to.Kind() == reflect.Int && from.Kind() != reflect.Int {
				return nil, fmt.Errorf("expected integer, got %s", from)
			}
			return value, nil
		}
	})
	if err != nil {
		return config, fmt.Errorf("decode configuration: %w", err)
	}
	if err := config.validate(); err != nil {
		return config, err
	}
	return config, nil
}

func (c Config) validate() error {
	_, port, err := net.SplitHostPort(c.Server.Address)
	if err != nil {
		return fmt.Errorf("server.address must have the form host:port or :port: %w", err)
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("server.address port must be between 1 and 65535")
	}
	if c.Server.ReadTimeout <= 0 || c.Server.WriteTimeout <= 0 || c.Server.IdleTimeout <= 0 || c.Server.ShutdownTimeout <= 0 {
		return fmt.Errorf("server timeouts must be positive durations")
	}
	if strings.TrimSpace(c.Log.Filename) == "" {
		return fmt.Errorf("log.filename must not be empty")
	}
	if c.Log.MaxSize <= 0 {
		return fmt.Errorf("log.max_size must be positive")
	}
	if c.Log.MaxBackups < 0 || c.Log.MaxAge < 0 {
		return fmt.Errorf("log.max_backups and log.max_age must be nonnegative")
	}
	return nil
}
