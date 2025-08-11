package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	// Command line flags
	var (
		host    = flag.String("host", "", "Server hostname or IP address")
		port    = flag.Int("port", 22, "SSH port")
		verbose = flag.Bool("verbose", false, "Enable verbose output")
		help    = flag.Bool("help", false, "Show help message")
	)
	flag.Parse()

	if *help || *host == "" {
		showHelp()
		return
	}

	fmt.Printf("🔍 SSH Connection Troubleshooter\n")
	fmt.Printf("================================\n\n")

	// Get current public IP
	fmt.Printf("📍 Detecting your public IP address...\n")
	currentIP, err := getCurrentPublicIP()
	if err != nil {
		fmt.Printf("⚠️  Could not determine public IP: %v\n", err)
		currentIP = "unknown"
	} else {
		fmt.Printf("✓ Your public IP: %s\n\n", currentIP)
	}

	// Test basic connectivity
	fmt.Printf("🌐 Testing network connectivity to %s:%d...\n", *host, *port)
	connected := testConnectivity(*host, *port, *verbose)

	if connected {
		fmt.Printf("✅ Connection successful! SSH port is reachable.\n")
		fmt.Printf("   Issue may be with SSH authentication or configuration.\n\n")
	} else {
		fmt.Printf("❌ Connection failed - analyzing potential causes...\n\n")
		analyzeFail2banIssue(*host, *port, currentIP, *verbose)
	}

	// Show additional diagnostic steps
	showDiagnosticSteps(*host, *port, currentIP)
}

func showHelp() {
	fmt.Printf(`SSH Connection Troubleshooter

This tool helps diagnose SSH connection issues, particularly fail2ban IP bans.

Usage:
  %s -host <hostname> [options]

Options:
  -host string      Server hostname or IP address (required)
  -port int         SSH port (default: 22)
  -verbose          Enable verbose output
  -help             Show this help message

Examples:
  %s -host 5.78.75.76
  %s -host myserver.com -port 2222 -verbose

`, os.Args[0], os.Args[0], os.Args[0])
}

func getCurrentPublicIP() (string, error) {
	services := []string{
		"https://ipinfo.io/ip",
		"https://api.ipify.org",
		"https://checkip.amazonaws.com",
	}

	client := &http.Client{Timeout: 5 * time.Second}

	for _, service := range services {
		resp, err := client.Get(service)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				continue
			}
			ip := strings.TrimSpace(string(body))
			if net.ParseIP(ip) != nil {
				return ip, nil
			}
		}
	}

	return "", fmt.Errorf("could not determine public IP from any service")
}

func testConnectivity(host string, port int, verbose bool) bool {
	address := net.JoinHostPort(host, strconv.Itoa(port))

	if verbose {
		fmt.Printf("   Attempting TCP connection to %s...\n", address)
	}

	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		if verbose {
			fmt.Printf("   Connection failed: %v\n", err)
		}
		return false
	}
	defer conn.Close()

	if verbose {
		fmt.Printf("   TCP connection successful\n")
	}
	return true
}

func analyzeFail2banIssue(host string, port int, currentIP string, verbose bool) {
	fmt.Printf("🕵️  Analyzing connection failure...\n\n")

	fmt.Printf("Most likely cause: FAIL2BAN IP BAN\n")
	fmt.Printf("================================\n")
	fmt.Printf("Since you mentioned this was working before and suddenly stopped,\n")
	fmt.Printf("your IP (%s) has likely been banned by fail2ban due to:\n", currentIP)
	fmt.Printf("• Multiple failed SSH login attempts\n")
	fmt.Printf("• Automated brute force detection\n")
	fmt.Printf("• Dynamic IP change that triggered security rules\n\n")

	fmt.Printf("Other possible causes:\n")
	fmt.Printf("• SSH service stopped running\n")
	fmt.Printf("• Firewall (UFW/iptables) blocking the port\n")
	fmt.Printf("• Server is down or unreachable\n")
	fmt.Printf("• Network routing issues\n\n")
}

func showDiagnosticSteps(host string, port int, currentIP string) {
	fmt.Printf("🔧 DIAGNOSTIC STEPS\n")
	fmt.Printf("==================\n\n")

	fmt.Printf("If you have console/VNC access to the server:\n\n")

	fmt.Printf("1. Check SSH service status:\n")
	fmt.Printf("   sudo systemctl status ssh\n")
	fmt.Printf("   sudo systemctl status sshd\n\n")

	fmt.Printf("2. Check fail2ban status:\n")
	fmt.Printf("   sudo systemctl status fail2ban\n")
	fmt.Printf("   sudo fail2ban-client status\n")
	fmt.Printf("   sudo fail2ban-client status sshd\n\n")

	fmt.Printf("3. Check if your IP is banned:\n")
	fmt.Printf("   sudo fail2ban-client get sshd banip\n")
	fmt.Printf("   sudo fail2ban-client get sshd banip | grep %s\n\n", currentIP)

	fmt.Printf("4. Check recent authentication failures:\n")
	fmt.Printf("   sudo journalctl -u ssh --since \"1 hour ago\" | grep Failed\n")
	fmt.Printf("   sudo journalctl -u fail2ban --since \"1 hour ago\"\n\n")

	fmt.Printf("🚨 TO FIX FAIL2BAN BAN:\n")
	fmt.Printf("======================\n\n")

	fmt.Printf("1. Unban your IP:\n")
	fmt.Printf("   sudo fail2ban-client set sshd unbanip %s\n\n", currentIP)

	fmt.Printf("2. If fail2ban seems stuck, restart it:\n")
	fmt.Printf("   sudo systemctl restart fail2ban\n\n")

	fmt.Printf("3. Verify the unban worked:\n")
	fmt.Printf("   sudo fail2ban-client get sshd banip | grep %s\n", currentIP)
	fmt.Printf("   (should return nothing if unbanned)\n\n")

	fmt.Printf("4. Test connection again:\n")
	fmt.Printf("   %s -host %s -port %d\n\n", os.Args[0], host, port)

	fmt.Printf("📞 ALTERNATIVE ACCESS METHODS:\n")
	fmt.Printf("==============================\n")
	fmt.Printf("• Console access through hosting provider\n")
	fmt.Printf("• VNC/remote desktop if configured\n")
	fmt.Printf("• Different public IP (mobile hotspot, VPN)\n")
	fmt.Printf("• Ask someone else to SSH in and unban your IP\n\n")

	fmt.Printf("💡 PREVENTION TIPS:\n")
	fmt.Printf("===================\n")
	fmt.Printf("• Use SSH keys instead of passwords\n")
	fmt.Printf("• Whitelist your static IP in fail2ban\n")
	fmt.Printf("• Increase fail2ban thresholds if too aggressive\n")
	fmt.Printf("• Monitor fail2ban logs regularly\n")
}
