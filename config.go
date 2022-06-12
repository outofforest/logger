package logger

import (
	"os"

	"github.com/pkg/errors"
	"github.com/ridge/must"
	"github.com/spf13/pflag"
)

// Format defines the format of log output
type Format string

const (
	// FormatConsole causes logs to be printed in console format delivered by zap
	FormatConsole Format = "console"

	// FormatJSON causes logs to be printed in JSON format delivered by zap
	FormatJSON Format = "json"

	// FormatYAML causes logs to be printed in YAML
	FormatYAML Format = "yaml"
)

// Config stores configuration of the logger
type Config struct {
	// Format defines the format of log output
	Format Format

	// Verbose turns on verbose logging
	Verbose bool
}

// ToolDefaultConfig stores handy default configuration used by tools run manually by humans
var ToolDefaultConfig = Config{
	Format:  FormatYAML,
	Verbose: false,
}

// ServiceDefaultConfig stores handy default configuration used by services
var ServiceDefaultConfig = Config{
	Format:  FormatJSON,
	Verbose: true,
}

var validFormats = map[Format]bool{
	FormatConsole: true,
	FormatJSON:    true,
	FormatYAML:    true,
}

// ConfigureWithCLI configures logger based on CLI flags
func ConfigureWithCLI(defaultConfig Config) Config {
	flags := pflag.NewFlagSet("logger", pflag.ContinueOnError)
	flags.ParseErrorsWhitelist.UnknownFlags = true
	AddFlags(defaultConfig, flags)
	// Dummy flag to turn off printing usage of this flag set
	flags.BoolP("help", "h", false, "")

	_ = flags.Parse(os.Args[1:])

	defaultConfig.Format = Format(must.String(flags.GetString("log-format")))
	defaultConfig.Verbose = must.Bool(flags.GetBool("verbose"))
	if !validFormats[defaultConfig.Format] {
		panic(errors.Errorf("incorrect logging format %s", defaultConfig.Format))
	}

	return defaultConfig
}

// Flags returns new flag set preconfigured with logger-specific options
func Flags(defaultConfig Config, name string) *pflag.FlagSet {
	flags := pflag.NewFlagSet(name, pflag.ContinueOnError)
	AddFlags(defaultConfig, flags)
	return flags
}

// AddFlags adds flags defined by logger
func AddFlags(defaultConfig Config, flags *pflag.FlagSet) {
	flags.String("log-format", string(defaultConfig.Format), "Format of log output: console | json")
	flags.BoolP("verbose", "v", defaultConfig.Verbose, "Turns on verbose logging")
}
