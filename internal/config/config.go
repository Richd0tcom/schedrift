package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type DatabaseConfig struct {
	Url          string `mapstructure:"url"`
	Driver       string `mapstructure:"driver"`
	Host         string `mapstructure:"host"`
	Port         string `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DatabaseName string `mapstructure:"database_name"`
	SSLMode      string `mapstructure:"sslmode"`
}

type SchemaConfig struct {
	IncludedSchemas []string `mapstructure:"included_schemas"`
	ExcludedSchemas []string `mapstructure:"excluded_schemas"`
	IncludedTables  []string `mapstructure:"included_tables"`
	ExcludedTables  []string `mapstructure:"excluded_tables"`
}

type OutputConfig struct {
	Format string `mapstructure:"format"`
	File   string `mapstructure:"file"`
}

type Config struct {
	DatabaseConfig DatabaseConfig `mapstructure:"database"`
	SchemaConfig   SchemaConfig   `mapstructure:"schema"`
	OutputConfig   OutputConfig   `mapstructure:"output"`
}

func SetupFlags(flags *pflag.FlagSet) {
	//DB
	flags.String("url", "", "Database connection URL")
	flags.String("host", "localhost", "Database host")
	flags.Int("port", 5432, "Database port")
	flags.String("user", "", "Database user")
	flags.String("password", "", "Database password")
	flags.String("dbname", "", "Database name")
	flags.String("sslmode", "prefer", "SSL mode (disable, prefer, require, verify-ca, verify-full)")

	//schema
	flags.StringSlice("include", []string{"public"}, "Schemas to include")
	flags.StringSlice("exclude", []string{}, "Schemas to exclude")
	flags.StringSlice("include-tables", []string{}, "Tables to include")
	flags.StringSlice("exclude-tables", []string{}, "Tables to exclude")

	//output
	flags.String("format", "sql", "output format (sql, json)")
	flags.String("output", "", "Output file (stdout if not specified)")
	flags.String("config", "", "Configuration file path")
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.sslmode", "prefer") //preffered ?
	v.SetDefault("schema.include", []string{"public"})
	v.SetDefault("output.format", "sql")
}

func LoadFromFlags(flags *pflag.FlagSet) (*Config, error) {
	v := viper.New()

	if err := v.BindPFlags(flags); err != nil {
		return nil, fmt.Errorf("error binding flags: %w", err)
	}

	v.SetEnvPrefix("SCHEMA_DRIFT")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	//checks if a config file is being used
	if cfgFile := v.GetString("config"); cfgFile != "" {
		v.SetConfigFile(cfgFile)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	setDefaults(v)

	// Override with environment variables for sensitive information
	if password := os.Getenv("PGPASSWORD"); password != "" {
		v.Set("password", password)
	}

	cfg := Config{}

	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	cfg.OutputConfig.File = v.GetString("config")

	return &cfg, nil

}
