package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/dean-jl/spf-flattener/internal/config"
	"github.com/dean-jl/spf-flattener/internal/porkbun"
	"github.com/spf13/cobra"
)

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Test the DNS API credentials for the domains specified in the config file.",
	Long: `Test the DNS API credentials for each domain listed in the config file.

This command loads the config file, then attempts to ping the DNS API for each domain using the provided credentials.
It reports success or failure for each domain, helping you verify that your API keys and secrets are correct.
`,
	Run: func(cmd *cobra.Command, args []string) {
		outputFile, _ := cmd.Flags().GetString("output")
		var outputBuilder strings.Builder

		startTime := time.Now()

		debugPrintlnf("[DEBUG] Starting ping command at %v\n", startTime)
		debugPrintln("[DEBUG] Loading config from:", cliConfig.ConfigPath)

		cfg, err := config.LoadConfig(cliConfig.ConfigPath)
		if err != nil {
			outputBuilder.WriteString(fmt.Sprintf("Failed to load config: %v\n", err))
			handleOutput(cmd, outputFile, &outputBuilder)
			log.Fatalf("Failed to load config: %v", err)
		}

		debugPrintlnf("[DEBUG] Loaded config with %d domains\n", len(cfg.Domains))
		verbosePrintlnf("[VERBOSE] Configuration loaded successfully from: %s\n", cliConfig.ConfigPath)

		if len(cfg.Domains) == 0 {
			outputBuilder.WriteString("No domains found in config file.\n")
			debugPrintln("[DEBUG] No domains found in config. Exiting.")
			handleOutput(cmd, outputFile, &outputBuilder)
			return
		}

		verbosePrintlnf("[VERBOSE] Testing API connectivity for %d domains\n", len(cfg.Domains))

		successCount := 0
		failureCount := 0

		for i, domain := range cfg.Domains {
			domainStartTime := time.Now()
			outputBuilder.WriteString(fmt.Sprintf("Pinging with credentials for domain: %s\n", domain.Name))

			verbosePrintlnf("[VERBOSE] [%d/%d] Processing domain: %s (provider: %s)\n",
				i+1, len(cfg.Domains), domain.Name, domain.Provider)
			debugPrintlnf("[DEBUG] Domain %s config: Provider=%s, TTL=%d\n",
				domain.Name, domain.Provider, domain.TTL)

			// Select DNS API client based on provider
			var apiClient interface{}
			switch strings.ToLower(domain.Provider) {
			case "porkbun":
				debugPrintlnf("[DEBUG] Creating Porkbun client for domain %s\n", domain.Name)
				apiClient = porkbun.NewClient(domain.ApiKey, domain.SecretKey, cliConfig.Debug)
			// Add other providers here (e.g., cloudflare, route53)
			default:
				msg := fmt.Sprintf("  No supported API client for provider: %s\n", domain.Provider)
				outputBuilder.WriteString(msg)
				verbosePrintlnf("[VERBOSE] Unsupported provider '%s' for domain %s\n", domain.Provider, domain.Name)
				failureCount++
				continue
			}

			if client, ok := apiClient.(porkbun.DNSAPIClient); ok {
				debugPrintlnf("[DEBUG] Sending ping request for domain %s\n", domain.Name)
				resp, err := client.Ping()
				domainDuration := time.Since(domainStartTime)

				if err != nil {
					msg := fmt.Sprintf("  Ping failed for %s: %v\n", domain.Name, err)
					outputBuilder.WriteString(msg)
					verbosePrintlnf("[VERBOSE] Ping failed for %s after %v: %v\n",
						domain.Name, domainDuration, err)
					debugPrintlnf("[DEBUG] Ping error details for %s: %+v\n", domain.Name, err)
					failureCount++
					continue
				}

				debugPrintlnf("[DEBUG] Ping response for %s: Status=%s, YourIP=%s\n",
					domain.Name, resp.Status, resp.YourIP)

				if resp.Status == "SUCCESS" {
					msg := fmt.Sprintf("  Ping successful for %s! Your IP is %s\n", domain.Name, resp.YourIP)
					outputBuilder.WriteString(msg)
					verbosePrintlnf("[VERBOSE] Ping successful for %s in %v (IP: %s)\n",
						domain.Name, domainDuration, resp.YourIP)
					successCount++
				} else {
					msg := fmt.Sprintf("  Ping failed for %s: %s\n", domain.Name, resp.Status)
					outputBuilder.WriteString(msg)
					verbosePrintlnf("[VERBOSE] Ping failed for %s after %v: status=%s\n",
						domain.Name, domainDuration, resp.Status)
					failureCount++
				}
			} else {
				msg := fmt.Sprintf("  API client for provider %s does not support Ping\n", domain.Provider)
				outputBuilder.WriteString(msg)
				debugPrintlnf("[DEBUG] Provider %s client does not implement ping interface\n", domain.Provider)
				failureCount++
			}
		}

		totalDuration := time.Since(startTime)

		// Add summary
		outputBuilder.WriteString(fmt.Sprintf("\n=== Ping Summary ===\n"))
		outputBuilder.WriteString(fmt.Sprintf("Total Domains: %d\n", len(cfg.Domains)))
		outputBuilder.WriteString(fmt.Sprintf("Successful: %d\n", successCount))
		outputBuilder.WriteString(fmt.Sprintf("Failed: %d\n", failureCount))
		outputBuilder.WriteString(fmt.Sprintf("Total Duration: %v\n", totalDuration))

		verbosePrintlnf("[VERBOSE] Ping command completed in %v\n", totalDuration)
		debugPrintlnf("[DEBUG] Final stats - Success: %d, Failed: %d, Total: %d\n",
			successCount, failureCount, len(cfg.Domains))

		handleOutput(cmd, outputFile, &outputBuilder)
	},
}

func init() {
	pingCmd.Flags().String("output", "", "Write output to a specified file instead of stdout")
}
