package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/dean-jl/spf-flattener/internal/config"
	"github.com/dean-jl/spf-flattener/internal/porkbun"
	"github.com/dean-jl/spf-flattener/internal/processor"
	"github.com/dean-jl/spf-flattener/internal/spf"
	"github.com/spf13/cobra"
	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
)

func processConfig(cmd *cobra.Command) (*config.Config, string, bool, error) {
	populateConfigFromFlags(cmd)
	outputFile, _ := cmd.Flags().GetString("output")
	force, _ := cmd.Flags().GetBool("force")

	if cliConfig.Production {
		cliConfig.DryRun = false
	}

	cfg, err := config.LoadConfig(cliConfig.ConfigPath)
	if err != nil {
		return nil, "", false, fmt.Errorf("failed to load config at %s: %w", cliConfig.ConfigPath, err)
	}

	return cfg, outputFile, force, nil
}

func setupLogger() *slog.Logger {
	if cliConfig.Debug {
		logLevel := new(slog.LevelVar)
		logLevel.Set(slog.LevelDebug)
		return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))
	}
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
}

func printStatusMessages() {
	if cliConfig.DryRun {
		fmt.Println("DRY-RUN: No changes will be applied.")
	}
	verbosePrintln("[VERBOSE] Verbose output enabled.")
	debugPrintln("[DEBUG] Debug output enabled.")
}

func setupDNSProvider(cfg *config.Config) spf.DNSProvider {
	if len(cfg.DNSServers) > 0 {
		verbosePrintln("[VERBOSE] DNS servers being used:")
		debugPrintlnf("[DEBUG] Setting up custom DNS provider with %d servers\n", len(cfg.DNSServers))
		for i, s := range cfg.DNSServers {
			verbosePrintlnf("  - %s (%s)\n", s.Name, s.IP)
			debugPrintlnf("[DEBUG] DNS server %d: %s -> %s\n", i+1, s.Name, s.IP)
		}
		var servers []string
		for _, s := range cfg.DNSServers {
			ip := s.IP
			if !strings.Contains(ip, ":") {
				ip = ip + ":53"
			}
			servers = append(servers, ip)
		}
		verbosePrintlnf("[VERBOSE] Using custom DNS servers: %v\n", servers)
		debugPrintlnf("[DEBUG] Custom DNS provider created with servers: %+v\n", servers)
		return spf.NewCustomDNSProvider(servers)
	}

	verbosePrintln("[VERBOSE] Using system DNS resolver.")
	debugPrintln("[DEBUG] Using default system DNS provider.")
	return &spf.DefaultDNSProvider{}
}

func handleOutput(cmd *cobra.Command, outputFile string, finalOutput *strings.Builder) {
	if outputFile != "" {
		err := os.WriteFile(outputFile, []byte(finalOutput.String()), 0644)
		if err != nil {
			cmd.PrintErrf("Error writing to output file %s: %v\n", outputFile, err)
			return
		}
	} else {
		fmt.Print(finalOutput.String())
	}
}

var flattenCmd = &cobra.Command{
	Use:   "flatten",
	Short: "Flatten SPF records for all configured domains.",
	Long:  `...`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, outputFile, force, err := processConfig(cmd)
		if err != nil {
			cmd.PrintErrf("Error: %v\n", err)
			return
		}

		logger := setupLogger()
		printStatusMessages()

		dnsProvider := setupDNSProvider(cfg)

		// Create domain processor with business logic (available for future refactoring)
		_ = processor.NewDomainProcessor(dnsProvider, cliConfig.Debug, cliConfig.DryRun, cliConfig.SpfUnflat)

		// --- Main Processing Logic ---
		// Create worker pool with maximum of 5 concurrent domain processors
		const maxWorkers = 5
		sem := semaphore.NewWeighted(maxWorkers)
		ctx := context.Background()

		var wg sync.WaitGroup
		domainResults := make(chan string, len(cfg.Domains))

		for i, domain := range cfg.Domains {
			verbosePrintlnf("[VERBOSE] [%d/%d] Starting processing for domain: %s\n", i+1, len(cfg.Domains), domain.Name)
			debugPrintlnf("[DEBUG] Domain %s config: Provider=%s, TTL=%d, SpfUnflat=%v\n",
				domain.Name, domain.Provider, domain.TTL, cliConfig.SpfUnflat)
			wg.Add(1)
			go func(d config.Domain, l *slog.Logger) {
				defer wg.Done()

				// Acquire semaphore token (blocks if max workers reached)
				if err := sem.Acquire(ctx, 1); err != nil {
					domainResults <- fmt.Sprintf("Error acquiring worker for domain %s: %v\n", d.Name, err)
					return
				}
				defer sem.Release(1)

				// Create rate limiter for API calls (2 requests per second with burst of 1)
				limiter := rate.NewLimiter(rate.Limit(2.0), 1)

				var resultBuf strings.Builder
				domainLogger := l.With("domain", d.Name)

				client := porkbun.NewClient(d.ApiKey, d.SecretKey, cliConfig.Debug)
				spfLookupName := d.Name
				if cliConfig.SpfUnflat {
					spfLookupName = "spf-unflat." + d.Name
				}

				originalSPF, flattenedSPF, err := spf.FlattenSPF(ctx, spfLookupName, dnsProvider)
				if err != nil {
					resultBuf.WriteString("\n===== Error processing domain: ")
					resultBuf.WriteString(d.Name)
					resultBuf.WriteString(" \n\n")
					resultBuf.WriteString("Error: ")
					resultBuf.WriteString(err.Error())
					resultBuf.WriteString("\n")
					domainResults <- resultBuf.String()
					return
				}

				existingRecordsResp, err := client.RetrieveRecords(d.Name)
				if err != nil {
					resultBuf.WriteString("\n===== Error processing domain: ")
					resultBuf.WriteString(d.Name)
					resultBuf.WriteString(" \n\n")
					resultBuf.WriteString("Error retrieving existing records: ")
					resultBuf.WriteString(err.Error())
					resultBuf.WriteString("\n")
					domainResults <- resultBuf.String()
					return
				}

				existingSPFTXTRecords := make(map[string]string)
				for _, record := range existingRecordsResp.Records {
					if record.Type != "TXT" {
						continue
					}
					recordName := strings.TrimSuffix(record.Name, ".")
					if recordName == d.Name {
						if strings.HasPrefix(record.Content, "v=spf1") {
							existingSPFTXTRecords[recordName] = record.Content
						}
					} else if strings.HasPrefix(recordName, "spf") {
						existingSPFTXTRecords[recordName] = record.Content
					}
				}

				currentAggregate := aggregateCurrentSPF(existingSPFTXTRecords, d.Name)

				// --- Change Detection ---
				recordsChanged := false
				changeSummary := ""

				var chainedRecords map[string]string // Declared here

				if currentAggregate == "(No valid SPF record found on root)" {
					recordsChanged = true
					chainedRecords = spf.SplitAndChainSPF(flattenedSPF, d.Name) // Assigned here
					newSet := spf.ExtractMechanismSet(flattenedSPF)
					var added []string
					for mech := range newSet {
						added = append(added, mech)
					}
					sort.Strings(added)
					if len(added) > 0 {
						changeSummary = "Added: " + strings.Join(added, ", ") + ". "
					}
				} else {
					normalizedOld, _ := spf.NormalizeSPF(currentAggregate)
					normalizedNew, _ := spf.NormalizeSPF(flattenedSPF)
					if normalizedOld != normalizedNew {
						recordsChanged = true
						chainedRecords = spf.SplitAndChainSPF(flattenedSPF, d.Name) // Assigned here
						oldSet := spf.ExtractMechanismSet(currentAggregate)
						newSet := spf.ExtractMechanismSet(flattenedSPF)
						var added, removed []string
						for mech := range newSet {
							if _, found := oldSet[mech]; !found {
								added = append(added, mech)
							}
						}
						for mech := range oldSet {
							if _, found := newSet[mech]; !found {
								removed = append(removed, mech)
							}
						}
						sort.Strings(added)
						sort.Strings(removed)
						if len(added) > 0 {
							changeSummary += "Added: " + strings.Join(added, ", ") + ". "
						}
						if len(removed) > 0 {
							changeSummary += "Removed: " + strings.Join(removed, ", ") + ". "
						}
					}
				}

				// Force flag overrides change detection
				if force && !recordsChanged {
					recordsChanged = true
					chainedRecords = spf.SplitAndChainSPF(flattenedSPF, d.Name)
					changeSummary = "No functional change to SPF mechanisms (forced update)."
				} else if changeSummary == "" {
					changeSummary = "No functional change to SPF mechanisms."
				}

				// --- Report Generation ---
				resultBuf.WriteString("\n===== Processing domain: ")
				resultBuf.WriteString(d.Name)
				resultBuf.WriteString(" \n\n")
				resultBuf.WriteString("---")
				resultBuf.WriteString(" SPF Summary ---")
				resultBuf.WriteString("\n\n")
				resultBuf.WriteString("Current Aggregate SPF:\n")
				resultBuf.WriteString(currentAggregate)
				resultBuf.WriteString("\n\n")
				resultBuf.WriteString("New Flattened Aggregate SPF:\n")
				resultBuf.WriteString(flattenedSPF)
				resultBuf.WriteString("\n\n")
				resultBuf.WriteString("Original SPF (unflattened):\n")
				resultBuf.WriteString(originalSPF)
				resultBuf.WriteString("\n\n")
				resultBuf.WriteString("---")
				resultBuf.WriteString(" Aggregate SPF Changes ---")
				resultBuf.WriteString("\n\n")
				resultBuf.WriteString(changeSummary)
				resultBuf.WriteString("\n\n")

				if recordsChanged {
					// chainedRecords is already defined and populated above
					resultBuf.WriteString("---")
					resultBuf.WriteString(" DNS TXT Records To Be Added/Changed ---")
					resultBuf.WriteString("\n\n")
					var sortedKeys []string
					// Add root domain first if it exists
					if _, ok := chainedRecords[d.Name]; ok {
						sortedKeys = append(sortedKeys, d.Name)
					}
					// Then add sorted spfN records
					var spfKeys []string
					for k := range chainedRecords {
						if k != d.Name {
							spfKeys = append(spfKeys, k)
						}
					}
					sort.Strings(spfKeys)
					sortedKeys = append(sortedKeys, spfKeys...)

					for _, key := range sortedKeys {
						resultBuf.WriteString("Record: ")
						resultBuf.WriteString(key)
						resultBuf.WriteString("\nValue: ")
						resultBuf.WriteString(chainedRecords[key])
						resultBuf.WriteString("\n\n")
					}
				}

				if !cliConfig.DryRun && recordsChanged {
					domainLogger.Info("SPF record changes detected, updating DNS records.")

					// Delete old, obsolete spfN records
					for name := range existingSPFTXTRecords {
						if strings.HasPrefix(name, "spf") && strings.Contains(name, d.Name) {
							if cliConfig.SpfUnflat && name == "spf-unflat."+d.Name {
								continue // Skip deletion of the spf-unflat source record
							}
							isStale := true
							for i := 0; ; i++ {
								spfName := "spf" + strconv.Itoa(i) + "." + d.Name
								if _, ok := chainedRecords[spfName]; !ok && name != spfName {
									continue
								}
								if name == spfName {
									isStale = false
									break
								}
								if _, ok := chainedRecords[spfName]; !ok {
									break
								}
							}

							if isStale {
								for _, rec := range existingRecordsResp.Records {
									if strings.TrimSuffix(rec.Name, ".") == name {
										limiter.Wait(ctx) // Rate limiting
										_, err := client.DeleteRecord(d.Name, rec.ID)
										if err != nil {
											domainLogger.Error("Failed to delete stale SPF record", "record", name, "error", err)
										} else {
											domainLogger.Info("Deleted stale SPF record", "record", name)
										}
										break
									}
								}
							}
						}
					}

					// Create/Update all SPF records (including main domain and spfN subdomains)
					if cliConfig.Debug {
						fmt.Printf("[DEBUG] chainedRecords contents:\n")
						for name, content := range chainedRecords {
							fmt.Printf("[DEBUG]   %s = %s\n", name, content)
						}
					}
					for name, content := range chainedRecords {
						// Extract the hostname for Porkbun API
						var hostName string
						if name == d.Name {
							// Root domain - use empty string
							hostName = ""
						} else {
							// Subdomain - extract just the subdomain part (e.g., "spf0" from "spf0.domain.com")
							hostName = strings.TrimSuffix(name, "."+d.Name)
						}

						var existingID string
						for _, rec := range existingRecordsResp.Records {
							if strings.TrimSuffix(rec.Name, ".") == name && rec.Type == "TXT" {
								existingID = rec.ID
								if cliConfig.Debug {
									fmt.Printf("[DEBUG] Found existing TXT record: name='%s', id=%s, type=%s\n", rec.Name, rec.ID, rec.Type)
								}
								break
							}
						}

						if existingID != "" {
							// Check if the existing record has the correct hostname
							var existingRecordHostname string
							for _, rec := range existingRecordsResp.Records {
								if rec.ID == existingID {
									existingRecordHostname = strings.TrimSuffix(rec.Name, ".")
									break
								}
							}

							// If hostname matches what we want, update content; otherwise delete and recreate
							if existingRecordHostname == name {
								if cliConfig.Debug {
									fmt.Printf("[DEBUG] Updating record: domain=%s, recordID=%s, hostName='%s', content='%s'\n", d.Name, existingID, hostName, content)
								}
								limiter.Wait(ctx) // Rate limiting
								_, err := client.UpdateRecordWithDetails(d.Name, existingID, hostName, "TXT", content, strconv.Itoa(d.TTL), "", "")
								if err != nil {
									if name == d.Name {
										domainLogger.Error("Failed to update main SPF record", "error", err)
									} else {
										domainLogger.Error("Failed to update SPF record", "record", name, "error", err)
									}
								} else {
									if name == d.Name {
										domainLogger.Info("Updated main SPF record")
									} else {
										domainLogger.Info("Updated SPF record", "record", name)
									}
								}
							} else {
								// Hostname mismatch - delete old record and create new one
								if cliConfig.Debug {
									fmt.Printf("[DEBUG] Hostname mismatch, deleting record: domain=%s, recordID=%s\n", d.Name, existingID)
								}
								limiter.Wait(ctx) // Rate limiting
								_, err := client.DeleteRecord(d.Name, existingID)
								if err != nil {
									domainLogger.Error("Failed to delete mismatched SPF record", "record", existingRecordHostname, "error", err)
								} else {
									domainLogger.Info("Deleted mismatched SPF record", "record", existingRecordHostname)
								}

								// Create new record with correct hostname
								if cliConfig.Debug {
									fmt.Printf("[DEBUG] Creating new record: domain=%s, hostName='%s', content='%s'\n", d.Name, hostName, content)
								}
								limiter.Wait(ctx) // Rate limiting
								_, err = client.CreateRecord(d.Name, hostName, "TXT", content, d.TTL)
								if err != nil {
									if name == d.Name {
										domainLogger.Error("Failed to create main SPF record", "error", err)
									} else {
										domainLogger.Error("Failed to create SPF record", "record", name, "error", err)
									}
								} else {
									if name == d.Name {
										domainLogger.Info("Created main SPF record")
									} else {
										domainLogger.Info("Created SPF record", "record", name)
									}
								}
							}
						} else {
							if cliConfig.Debug {
								fmt.Printf("[DEBUG] Creating record: domain=%s, hostName='%s', content='%s'\n", d.Name, hostName, content)
							}
							limiter.Wait(ctx) // Rate limiting
							_, err := client.CreateRecord(d.Name, hostName, "TXT", content, d.TTL)
							if err != nil {
								if name == d.Name {
									domainLogger.Error("Failed to create main SPF record", "error", err)
								} else {
									domainLogger.Error("Failed to create SPF record", "record", name, "error", err)
								}
							} else {
								if name == d.Name {
									domainLogger.Info("Created main SPF record")
								} else {
									domainLogger.Info("Created SPF record", "record", name)
								}
							}
						}
					}
					resultBuf.WriteString("\nSPF records updated in production mode.\n")
				} else if cliConfig.DryRun && recordsChanged {
					resultBuf.WriteString("\nSPF records would be updated in production mode.\n")
				} else {
					resultBuf.WriteString("\nSPF records are already up to date. No changes needed.\n")
				}

				resultBuf.WriteString("\n---")
				resultBuf.WriteString("\n" + client.Attribution())
				resultBuf.WriteString("\n---\n")

				domainResults <- resultBuf.String()
			}(domain, logger)
		}

		wg.Wait()
		close(domainResults)

		// --- Output Handling ---
		var finalOutput strings.Builder
		for result := range domainResults {
			finalOutput.WriteString(result)
		}

		handleOutput(cmd, outputFile, &finalOutput)
	},
}

func aggregateCurrentSPF(records map[string]string, domain string) string {
	var aggregatedMechanisms []string
	seenIncludes := make(map[string]bool)

	var processRecord func(string)
	processRecord = func(recordContent string) {
		parts := strings.Fields(recordContent)
		for _, part := range parts {
			if strings.HasPrefix(part, "v=spf1") || strings.HasSuffix(part, "all") {
				continue
			}

			if strings.HasPrefix(part, "include:") {
				includeDomain := strings.TrimPrefix(part, "include:")
				if !seenIncludes[includeDomain] {
					seenIncludes[includeDomain] = true
					if nextRecord, ok := records[includeDomain]; ok {
						processRecord(nextRecord)
					} else {
						aggregatedMechanisms = append(aggregatedMechanisms, part)
					}
				}
			} else {
				aggregatedMechanisms = append(aggregatedMechanisms, part)
			}
		}
	}

	if rootSPF, ok := records[domain]; ok {
		processRecord(rootSPF)
	} else {
		return "(No valid SPF record found on root)"
	}

	normalized, err := spf.NormalizeSPF("v=spf1 " + strings.Join(aggregatedMechanisms, " ") + " ~all")
	if err != nil {
		return "(Could not normalize existing record)"
	}
	return normalized
}

func init() {
	flattenCmd.Flags().Bool("dry-run", true, "Simulate changes without applying them") // Default to true for safety
	flattenCmd.Flags().String("output", "", "Write final reports to a specified file instead of stdout")
	flattenCmd.Flags().Bool("production", false, "Enable production mode (live DNS updates)") // New flag
	flattenCmd.Flags().Bool("force", false, "Force update DNS records regardless of changes") // Force flag
}
