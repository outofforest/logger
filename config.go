package logger

import (
	"fmt"
	"os"

	"github.com/ridge/must"
	"github.com/spf13/pflag"
)

// Format defines the format of log output
type Format string

const (
	// FormatConsole causes logs to be printed in human-readable form
	FormatConsole Format = "console"

	// FormatJSON causes logs to be printed in JSON
	FormatJSON Format = "json"
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
	Format:  FormatConsole,
	Verbose: false,
}

// ServiceDefaultConfig stores handy default configuration used by services
var ServiceDefaultConfig = Config{
	Format:  FormatJSON,
	Verbose: true,
}

// ConfigureWithCLI configures logger based on CLI flags
func ConfigureWithCLI(defaultConfig Config) Config {
	var format string
	flags := pflag.NewFlagSet("logger", pflag.ContinueOnError)
	flags.ParseErrorsWhitelist.UnknownFlags = true
	AddFlags(defaultConfig, flags)
	// Dummy flag to turn off printing usage of this flag set
	flags.BoolP("help", "h", false, "")

	_ = flags.Parse(os.Args[1:])

	defaultConfig.Format = Format(must.String(flags.GetString("log-format")))
	defaultConfig.Verbose = must.Bool(flags.GetBool("verbose"))
	if defaultConfig.Format != FormatConsole && defaultConfig.Format != FormatJSON {
		panic(fmt.Errorf("incorrect logging format %s", format))
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
