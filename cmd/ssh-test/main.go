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
		host           = flag.String("host", "", "SSH server hostname or IP address")
		port           = flag.Int("port", 22, "SSH server port")
		appUser        = flag.String("app-user", "pocketbase", "Application username")
		rootUser       = flag.String("root-user", "root", "Root username")
		asRoot         = flag.Bool("root", false, "Connect as root user")
		keyPath        = flag.String("key", "", "Path to private key file")
		useAgent       = flag.Bool("agent", true, "Use SSH agent for authentication")
		verbose        = flag.Bool("v", false, "Verbose output")
		securityLocked = flag.Bool("security-locked", false, "Server has security lockdown applied")

		// Basic operations
		test      = flag.Bool("test", false, "Test SSH connection")
		testBoth  = flag.Bool("test-both", false, "Test both root and app user connections")
		acceptKey = flag.Bool("accept-key", false, "Pre-accept host key")
		fix       = flag.Bool("fix", false, "Attempt to fix common issues")

		// General diagnostics
		diagnose     = flag.Bool("diagnose", false, "Run connection diagnostics")
		troubleshoot = flag.Bool("troubleshoot", false, "Run comprehensive troubleshooting")

		// Pre-security operations
		preSecurity = flag.Bool("pre-security", false, "Run pre-security-lockdown diagnostics and preparation")
		setup       = flag.Bool("setup", false, "Auto-setup prerequisites for security lockdown")

		// Post-security operations
		postSecurity = flag.Bool("post-security", false, "Run post-security-lockdown diagnostics")
		autoFix      = flag.Bool("auto-fix", false, "Attempt automatic fixes during post-security diagnostics")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "SSH Connection Testing and Troubleshooting Tool\n\n")
		fmt.Fprintf(os.Stderr, "This tool helps with SSH diagnostics, pre-security setup, and post-security troubleshooting.\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])

		fmt.Fprintf(os.Stderr, "Required:\n")
		fmt.Fprintf(os.Stderr, "  -host string\n        SSH server hostname or IP address\n\n")

		fmt.Fprintf(os.Stderr, "Server Configuration:\n")
		fmt.Fprintf(os.Stderr, "  -port int\n        SSH server port (default 22)\n")
		fmt.Fprintf(os.Stderr, "  -app-user string\n        Application username (default \"pocketbase\")\n")
		fmt.Fprintf(os.Stderr, "  -root-user string\n        Root username (default \"root\")\n")
		fmt.Fprintf(os.Stderr, "  -key string\n        Path to private key file\n")
		fmt.Fprintf(os.Stderr, "  -agent\n        Use SSH agent for authentication (default true)\n")
		fmt.Fprintf(os.Stderr, "  -security-locked\n        Server has security lockdown applied\n")
		fmt.Fprintf(os.Stderr, "  -v    Verbose output\n\n")

		fmt.Fprintf(os.Stderr, "Basic Operations:\n")
		fmt.Fprintf(os.Stderr, "  -test\n        Test SSH connection\n")
		fmt.Fprintf(os.Stderr, "  -test-both\n        Test both root and app user connections\n")
		fmt.Fprintf(os.Stderr, "  -root\n        Connect as root user (for -test)\n")
		fmt.Fprintf(os.Stderr, "  -accept-key\n        Pre-accept host key to fix 'key is unknown' error\n")
		fmt.Fprintf(os.Stderr, "  -fix  Attempt to fix common SSH issues\n\n")

		fmt.Fprintf(os.Stderr, "Diagnostics:\n")
		fmt.Fprintf(os.Stderr, "  -diagnose\n        Run connection diagnostics\n")
		fmt.Fprintf(os.Stderr, "  -troubleshoot\n        Run comprehensive troubleshooting\n\n")

		fmt.Fprintf(os.Stderr, "Pre-Security Lockdown:\n")
		fmt.Fprintf(os.Stderr, "  -pre-security\n        Run pre-security-lockdown diagnostics and preparation\n")
		fmt.Fprintf(os.Stderr, "  -setup\n        Auto-setup prerequisites for security lockdown\n\n")

		fmt.Fprintf(os.Stderr, "Post-Security Lockdown:\n")
		fmt.Fprintf(os.Stderr, "  -post-security\n        Run post-security-lockdown diagnostics\n")
		fmt.Fprintf(os.Stderr, "  -auto-fix\n        Attempt automatic fixes during post-security diagnostics\n\n")

		fmt.Fprintf(os.Stderr, "Examples:\n\n")

		fmt.Fprintf(os.Stderr, "  Basic connection test:\n")
		fmt.Fprintf(os.Stderr, "    %s -host 91.99.196.153 -test\n\n", os.Args[0])

		fmt.Fprintf(os.Stderr, "  Pre-security setup and preparation:\n")
		fmt.Fprintf(os.Stderr, "    %s -host 91.99.196.153 -pre-security -setup\n\n", os.Args[0])

		fmt.Fprintf(os.Stderr, "  Post-security diagnostics on locked server:\n")
		fmt.Fprintf(os.Stderr, "    %s -host 91.99.196.153 -post-security -security-locked\n\n", os.Args[0])

		fmt.Fprintf(os.Stderr, "  Post-security with auto-fix:\n")
		fmt.Fprintf(os.Stderr, "    %s -host 91.99.196.153 -post-security -auto-fix -security-locked\n\n", os.Args[0])

		fmt.Fprintf(os.Stderr, "  Comprehensive troubleshooting:\n")
		fmt.Fprintf(os.Stderr, "    %s -host 91.99.196.153 -troubleshoot\n\n", os.Args[0])

		fmt.Fprintf(os.Stderr, "  Pre-accept host key:\n")
		fmt.Fprintf(os.Stderr, "    %s -host 91.99.196.153 -accept-key\n\n", os.Args[0])

		fmt.Fprintf(os.Stderr, "  Test both connections:\n")
		fmt.Fprintf(os.Stderr, "    %s -host 91.99.196.153 -test-both\n\n", os.Args[0])

		fmt.Fprintf(os.Stderr, "Workflow:\n")
		fmt.Fprintf(os.Stderr, "  1. Use -pre-security -setup to prepare server for lockdown\n")
		fmt.Fprintf(os.Stderr, "  2. Apply security lockdown (disable root SSH)\n")
		fmt.Fprintf(os.Stderr, "  3. Use -post-security -security-locked to verify lockdown\n\n")
	}

	flag.Parse()

	if *host == "" {
		fmt.Fprintf(os.Stderr, "Error: host is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Create server model
	server := &models.Server{
		Host:           *host,
		Port:           *port,
		AppUsername:    *appUser,
		RootUsername:   *rootUser,
		UseSSHAgent:    *useAgent,
		ManualKeyPath:  *keyPath,
		SecurityLocked: *securityLocked,
	}

	if *verbose {
		fmt.Printf("Server configuration:\n")
		fmt.Printf("  Host: %s:%d\n", server.Host, server.Port)
		fmt.Printf("  App User: %s\n", server.AppUsername)
		fmt.Printf("  Root User: %s\n", server.RootUsername)
		fmt.Printf("  Use SSH Agent: %v\n", server.UseSSHAgent)
		fmt.Printf("  Manual Key: %s\n", server.ManualKeyPath)
		fmt.Printf("  Security Locked: %v\n", server.SecurityLocked)
		fmt.Printf("  Connect as root: %v\n", *asRoot)
		fmt.Printf("\n")
	}

	// Determine what actions to perform
	hasAction := *diagnose || *postSecurity || *preSecurity || *fix || *acceptKey || *test || *testBoth || *troubleshoot
	if !hasAction {
		// Default action: basic diagnostics
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

	// Run pre-security diagnostics and setup if requested
	if *preSecurity {
		fmt.Printf("🔧 Pre-Security Lockdown Mode\n\n")

		result := RunPreSecurityDiagnostics(server, *setup)

		if !result.ReadyForLockdown {
			exitCode = 1
		}

		// Show summary if verbose
		if *verbose {
			summary, err := GetPreSecuritySummary(server)
			if err != nil {
				fmt.Printf("⚠️  Could not generate summary: %v\n", err)
			} else {
				fmt.Printf("\n%s\n", summary)
			}
		}

		fmt.Println()
	}

	// Run post-security diagnostics if requested
	if *postSecurity {
		fmt.Printf("🔒 Post-Security Lockdown Mode\n\n")

		result := RunPostSecurityDiagnostics(server, *autoFix)

		if !result.OverallPass {
			exitCode = 1
		}

		// Show summary if verbose
		if *verbose {
			summary, err := GetPostSecuritySummary(server)
			if err != nil {
				fmt.Printf("⚠️  Could not generate summary: %v\n", err)
			} else {
				fmt.Printf("\n%s\n", summary)
			}
		}

		fmt.Println()
	}

	// Run general diagnostics if requested
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

	// Run comprehensive troubleshooting if requested
	if *troubleshoot {
		fmt.Printf("Running comprehensive troubleshooting for %s:%d...\n\n", server.Host, server.Port)

		// First run general diagnostics
		summary, err := ssh.GetConnectionSummary(server, *asRoot)
		if err != nil {
			fmt.Printf("❌ Failed to run general diagnostics: %v\n", err)
			exitCode = 1
		} else {
			fmt.Print(summary)
		}

		// If security-locked, also run post-security diagnostics
		if *securityLocked {
			fmt.Printf("\nRunning additional post-security diagnostics...\n\n")

			result := RunPostSecurityDiagnostics(server, false) // No auto-fix in troubleshoot mode
			if !result.OverallPass {
				exitCode = 1
			}
		} else {
			// If not security-locked, run pre-security checks
			fmt.Printf("\nRunning pre-security readiness checks...\n\n")

			summary, err := GetPreSecuritySummary(server)
			if err != nil {
				fmt.Printf("❌ Failed to run pre-security checks: %v\n", err)
				exitCode = 1
			} else {
				fmt.Print(summary)
			}
		}

		fmt.Println()
	}

	// Test single connection if requested
	if *test {
		username := server.AppUsername
		if *asRoot {
			username = server.RootUsername
		}

		fmt.Printf("Testing SSH connection to %s:%d as %s...\n", server.Host, server.Port, username)

		if *securityLocked && *asRoot {
			fmt.Printf("⚠️  Warning: Testing root connection on security-locked server (expected to fail)\n")
		}

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
		fmt.Println()
	}

	// Test both connections if requested
	if *testBoth {
		fmt.Printf("Testing both SSH connections to %s:%d...\n\n", server.Host, server.Port)

		// Test root connection
		fmt.Printf("Testing root connection (%s)...\n", server.RootUsername)
		rootManager, err := ssh.NewSSHManager(server, true)
		if err != nil {
			if *securityLocked {
				fmt.Printf("⚠️  Root connection failed (expected on security-locked server): %v\n", err)
			} else {
				fmt.Printf("❌ Root connection failed: %v\n", err)
				exitCode = 1
			}
		} else {
			defer rootManager.Close()
			if err := rootManager.TestConnection(); err != nil {
				if *securityLocked {
					fmt.Printf("⚠️  Root connection test failed (expected): %v\n", err)
				} else {
					fmt.Printf("❌ Root connection test failed: %v\n", err)
					exitCode = 1
				}
			} else {
				fmt.Printf("✅ Root connection successful\n")
			}
		}

		fmt.Println()

		// Test app user connection
		fmt.Printf("Testing app user connection (%s)...\n", server.AppUsername)
		appManager, err := ssh.NewSSHManager(server, false)
		if err != nil {
			fmt.Printf("❌ App user connection failed: %v\n", err)
			exitCode = 1
		} else {
			defer appManager.Close()
			if err := appManager.TestConnection(); err != nil {
				fmt.Printf("❌ App user connection test failed: %v\n", err)
				exitCode = 1
			} else {
				fmt.Printf("✅ App user connection successful\n")

				// Test sudo access for app user
				if *verbose {
					fmt.Printf("Testing sudo access...\n")
					_, err := appManager.ExecuteCommand("sudo -n systemctl --version")
					if err != nil {
						fmt.Printf("⚠️  Sudo access test failed: %v\n", err)
					} else {
						fmt.Printf("✅ Sudo access working\n")
					}
				}
			}
		}
		fmt.Println()
	}

	// Final summary and suggestions
	if exitCode != 0 {
		fmt.Printf("❌ Some operations failed.\n")
		if !*verbose {
			fmt.Printf("💡 Use -v for more details.\n")
		}

		// Provide context-aware suggestions
		if *securityLocked {
			fmt.Printf("ℹ️  Note: On security-locked servers, root SSH access is expected to be disabled.\n")
			if !*postSecurity {
				fmt.Printf("💡 Try: %s -host %s -post-security -security-locked\n", os.Args[0], *host)
			}
		} else {
			if !*preSecurity {
				fmt.Printf("💡 For pre-lockdown setup: %s -host %s -pre-security -setup\n", os.Args[0], *host)
			}
		}

		// Generic suggestions
		if !*acceptKey {
			fmt.Printf("💡 For host key issues: %s -host %s -accept-key\n", os.Args[0], *host)
		}
		if !*fix {
			fmt.Printf("💡 For common fixes: %s -host %s -fix\n", os.Args[0], *host)
		}
		if !*troubleshoot {
			fmt.Printf("💡 For comprehensive analysis: %s -host %s -troubleshoot\n", os.Args[0], *host)
		}
	} else {
		fmt.Printf("✅ All operations completed successfully!\n")

		// Provide workflow guidance
		if *preSecurity && !*securityLocked {
			fmt.Printf("🔧 Next step: Apply security lockdown to disable root SSH access.\n")
			fmt.Printf("🔒 After lockdown, test with: %s -host %s -post-security -security-locked\n", os.Args[0], *host)
		} else if *postSecurity && *securityLocked {
			fmt.Printf("🔒 Server is properly secured and app user access is working.\n")
		}
	}

	os.Exit(exitCode)
}
