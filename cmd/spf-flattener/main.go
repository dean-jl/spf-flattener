package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/dean-jl/spf-flattener/internal/config"
	"github.com/spf13/cobra"
)

// CLIConfig holds CLI flag values
type CLIConfig struct {
	SpfUnflat  bool
	ConfigPath string
	Debug      bool
	Verbose    bool
	Production bool
	DryRun     bool
	Aggregate  bool
}

var cliConfig = &CLIConfig{}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "spf-flattener",
	Short: "SPF Flattener is a CLI tool to flatten SPF records.",
	Long:  "A command-line tool to flatten SPF DNS records for multiple domains using the Porkbun API.",
	Run: func(cmd *cobra.Command, args []string) {
		// Main CLI logic goes here (currently empty for root command)
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&cliConfig.SpfUnflat, "spf-unflat", false, "Use spf-unflat.<domain> TXT record as source instead of main SPF record (preserves original unflattened SPF for future updates)")
	rootCmd.PersistentFlags().StringVar(&cliConfig.ConfigPath, "config", "config.yaml", "Path to configuration file")
	rootCmd.PersistentFlags().BoolVar(&cliConfig.Debug, "debug", false, "Enable debug output")
	rootCmd.PersistentFlags().BoolVar(&cliConfig.Verbose, "verbose", false, "Enable verbose output")
	rootCmd.AddCommand(pingCmd)
	rootCmd.AddCommand(flattenCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(importCmd)

	rootCmd.Version = config.Version
	rootCmd.SetHelpTemplate("SPF Flattener v" + config.Version + "\n\n{{.Long}}\n\nUsage:\n  {{.UseLine}}\n\nAvailable Commands:\n{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name \"help\"))}}  {{rpad .Name .NamePadding }} {{.Short}}\n{{end}}{{end}}\n\nFlags:\n{{.Flags.FlagUsages | trimTrailingWhitespaces}}\n\nUse \"{{.UseLine}} [command] --help\" for more information about a command.\n")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func logEvent(logger *log.Logger, level, message string, data map[string]string) {
	event := map[string]string{
		"level":   level,
		"message": message,
	}
	for k, v := range data {
		event[k] = v
	}
	jsonEvent, _ := json.Marshal(event)
	logger.Println(string(jsonEvent))
}

// populateConfigFromFlags updates cliConfig with current flag values
func populateConfigFromFlags(cmd *cobra.Command) {
	if dryRun, err := cmd.Flags().GetBool("dry-run"); err == nil {
		cliConfig.DryRun = dryRun
	}
	if production, err := cmd.Flags().GetBool("production"); err == nil {
		cliConfig.Production = production
	}
	if aggregate, err := cmd.Flags().GetBool("aggregate"); err == nil {
		cliConfig.Aggregate = aggregate
	}
}

func debugPrintln(a ...interface{}) {
	if cliConfig.Debug {
		fmt.Println(a...)
	}
}

// verbosePrintln prints verbose messages when verbose mode is enabled
func verbosePrintln(a ...interface{}) {
	if cliConfig.Verbose {
		fmt.Println(a...)
	}
}

// verbosePrintlnf prints formatted verbose messages when verbose mode is enabled
func verbosePrintlnf(format string, a ...interface{}) {
	if cliConfig.Verbose {
		fmt.Printf(format, a...)
	}
}

// debugPrintlnf prints formatted debug messages when debug mode is enabled
func debugPrintlnf(format string, a ...interface{}) {
	if cliConfig.Debug {
		fmt.Printf(format, a...)
	}
}

func main() {
	Execute()
}
