package main

import (
	"flag"
	"fmt"
	"os"

	"pb-deployer/internal/models"
	"pb-deployer/internal/ssh"
)

func main() {
	var (
		host      = flag.String("host", "", "SSH server hostname or IP address")
		port      = flag.Int("port", 22, "SSH server port")
		appUser   = flag.String("app-user", "pocketbase", "Application username")
		rootUser  = flag.String("root-user", "root", "Root username")
		asRoot    = flag.Bool("root", false, "Connect as root user")
		keyPath   = flag.String("key", "", "Path to private key file")
		useAgent  = flag.Bool("agent", true, "Use SSH agent for authentication")
		diagnose  = flag.Bool("diagnose", false, "Run connection diagnostics")
		fix       = flag.Bool("fix", false, "Attempt to fix common issues")
		acceptKey = flag.Bool("accept-key", false, "Pre-accept host key")
		test      = flag.Bool("test", false, "Test SSH connection")
		verbose   = flag.Bool("v", false, "Verbose output")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "SSH Connection Troubleshooting Tool\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Diagnose connection issues\n")
		fmt.Fprintf(os.Stderr, "  %s -host 91.99.196.153 -diagnose\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Pre-accept host key to fix 'key is unknown' error\n")
		fmt.Fprintf(os.Stderr, "  %s -host 91.99.196.153 -accept-key\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Test connection as root\n")
		fmt.Fprintf(os.Stderr, "  %s -host 91.99.196.153 -root -test\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Fix common issues and test\n")
		fmt.Fprintf(os.Stderr, "  %s -host 91.99.196.153 -fix -test\n\n", os.Args[0])
	}

	flag.Parse()

	if *host == "" {
		fmt.Fprintf(os.Stderr, "Error: host is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Create server model
	server := &models.Server{
		Host:          *host,
		Port:          *port,
		AppUsername:   *appUser,
		RootUsername:  *rootUser,
		UseSSHAgent:   *useAgent,
		ManualKeyPath: *keyPath,
	}

	if *verbose {
		fmt.Printf("Server configuration:\n")
		fmt.Printf("  Host: %s:%d\n", server.Host, server.Port)
		fmt.Printf("  App User: %s\n", server.AppUsername)
		fmt.Printf("  Root User: %s\n", server.RootUsername)
		fmt.Printf("  Use SSH Agent: %v\n", server.UseSSHAgent)
		fmt.Printf("  Manual Key: %s\n", server.ManualKeyPath)
		fmt.Printf("  Connect as root: %v\n", *asRoot)
		fmt.Printf("\n")
	}

	// Determine what actions to perform
	if !*diagnose && !*fix && !*acceptKey && !*test {
		// Default action: diagnose
		*diagnose = true
	}

	var exitCode int

	// Pre-accept host key if requested
	if *acceptKey {
		fmt.Printf("Pre-accepting host key for %s:%d...\n", server.Host, server.Port)
		if err := ssh.AcceptHostKey(server); err != nil {
			fmt.Printf("❌ Failed to pre-accept host key: %v\n", err)
			exitCode = 1
		} else {
			fmt.Printf("✅ Host key pre-accepted successfully\n")
		}
		fmt.Println()
	}

	// Fix common issues if requested
	if *fix {
		fmt.Printf("Attempting to fix common SSH issues...\n")
		results := ssh.FixCommonIssues(server)

		for _, result := range results {
			status := "✅"
			if result.Status == "error" {
				status = "❌"
				exitCode = 1
			} else if result.Status == "warning" {
				status = "⚠️"
			}

			fmt.Printf("%s %s: %s\n", status, result.Step, result.Message)
			if result.Details != "" && *verbose {
				fmt.Printf("   Details: %s\n", result.Details)
			}
		}
		fmt.Println()
	}

	// Run diagnostics if requested
	if *diagnose {
		fmt.Printf("Running SSH connection diagnostics for %s:%d...\n\n", server.Host, server.Port)

		summary, err := ssh.GetConnectionSummary(server, *asRoot)
		if err != nil {
			fmt.Printf("❌ Failed to run diagnostics: %v\n", err)
			exitCode = 1
		} else {
			fmt.Print(summary)
		}
		fmt.Println()
	}

	// Test connection if requested
	if *test {
		fmt.Printf("Testing SSH connection to %s:%d", server.Host, server.Port)
		if *asRoot {
			fmt.Printf(" as %s", server.RootUsername)
		} else {
			fmt.Printf(" as %s", server.AppUsername)
		}
		fmt.Printf("...\n")

		manager, err := ssh.NewSSHManager(server, *asRoot)
		if err != nil {
			fmt.Printf("❌ Failed to establish SSH connection: %v\n", err)
			exitCode = 1
		} else {
			defer manager.Close()

			// Test basic connectivity
			if err := manager.TestConnection(); err != nil {
				fmt.Printf("❌ SSH connection test failed: %v\n", err)
				exitCode = 1
			} else {
				fmt.Printf("✅ SSH connection test successful!\n")

				if *verbose {
					info := manager.GetConnectionInfo()
					fmt.Printf("Connection details:\n")
					for key, value := range info {
						fmt.Printf("  %s: %v\n", key, value)
					}
				}
			}
		}
	}

	if exitCode != 0 {
		fmt.Printf("\n❌ Some operations failed. Use -v for more details.\n")
	} else {
		fmt.Printf("\n✅ All operations completed successfully!\n")
	}

	os.Exit(exitCode)
}
