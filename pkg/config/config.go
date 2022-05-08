package config

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	KeyDsn    = "dsn"
	KeyPath   = "path"
	KeyFormat = "type"
	KeyTable  = "table"
)

const (
	EnvPrefix         = "GOMIGRATOR"
	DefaultConfigFile = "./.gomigrator.yml"
)

var cfg *Config

func init() {
	cfg = New()
}

type Config struct {
	v *viper.Viper
}

func New() *Config {
	cfg := new(Config)
	cfg.v = viper.New()
	cfg.init()
	return cfg
}

func (c *Config) init() {
	c.v.SetDefault(KeyDsn, "")
	c.v.SetDefault(KeyPath, "")
	c.v.SetDefault(KeyFormat, "")
	c.v.SetDefault(KeyTable, "dbmigrator_version")
}

func (c *Config) Load(file string, flags *pflag.FlagSet) error {
	if err := c.LoadFromEnv(); err != nil {
		return err
	}
	if file != "" {
		if err := c.LoadFromFile(file); err != nil {
			return err
		}
	}
	if err := c.LoadFromCommandLine(flags); err != nil {
		return err
	}
	return nil
}

func Load(file string, flags *pflag.FlagSet) error {
	return cfg.Load(file, flags)
}

func (c *Config) WriteConfig(file string) error {
	if err := c.v.SafeWriteConfigAs(file); err != nil {
		return err
	}
	return nil
}

func WriteConfig(file string) error {
	return cfg.WriteConfig(file)
}

func (c *Config) AddConfigPath(path string) {
	c.v.AddConfigPath(path)
}

func AddConfigPath(path string) {
	cfg.AddConfigPath(path)
}

func (c *Config) LoadFromEnv() error {
	c.v.SetEnvPrefix(EnvPrefix)
	c.v.AutomaticEnv()
	return nil
}

func (c *Config) LoadFromFile(file string) error {
	c.v.SetConfigFile(file)
	if err := c.v.ReadInConfig(); err != nil {
		return err
	}
	return nil
}

func LoadFromFile(file string) error {
	return cfg.LoadFromFile(file)
}

func (c *Config) LoadFromCommandLine(flagSet *pflag.FlagSet) error {
	if err := c.v.BindPFlags(flagSet); err != nil {
		return err
	}
	return nil
}

func LoadFromCommandLine(flagSet *pflag.FlagSet) error {
	return cfg.LoadFromCommandLine(flagSet)
}

func (c *Config) GetDsn() string {
	return c.v.GetString(KeyDsn)
}

func GetDsn() string {
	return cfg.GetDsn()
}

func (c *Config) GetPath() string {
	return c.v.GetString(KeyPath)
}

func GetPath() string {
	return cfg.GetPath()
}

func (c *Config) GetType() string {
	return c.v.GetString(KeyFormat)
}

func GetType() string {
	return cfg.GetType()
}

func (c *Config) GetTable() string {
	return c.v.GetString(KeyTable)
}

func GetTable() string {
	return cfg.GetTable()
}
