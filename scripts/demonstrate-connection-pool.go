package main

import (
	"fmt"
	"sync"
	"time"

	"pb-deployer/internal/models"
	"pb-deployer/internal/ssh"
)

func main() {
	fmt.Println("🔗 SSH Connection Pool Demonstration")
	fmt.Println("=====================================")

	// Example server configuration
	server := &models.Server{
		ID:           "demo-server",
		Name:         "Demo Server",
		Host:         "192.168.1.100", // Replace with your test server
		Port:         22,
		RootUsername: "root",
		AppUsername:  "pocketbase",
		UseSSHAgent:  true,
	}

	fmt.Printf("Target Server: %s (%s:%d)\n\n", server.Name, server.Host, server.Port)

	// Demonstration 1: Connection Pool vs Direct Connections
	fmt.Println("📊 DEMO 1: Performance Comparison")
	fmt.Println("----------------------------------")
	demonstratePerformance(server)

	// Demonstration 2: Connection Health Monitoring
	fmt.Println("\n🏥 DEMO 2: Health Monitoring")
	fmt.Println("-----------------------------")
	demonstrateHealthMonitoring(server)

	// Demonstration 3: Concurrent Operations
	fmt.Println("\n🚀 DEMO 3: Concurrent Operations")
	fmt.Println("---------------------------------")
	demonstrateConcurrentOperations(server)

	// Demonstration 4: Connection Recovery
	fmt.Println("\n🔄 DEMO 4: Connection Recovery")
	fmt.Println("------------------------------")
	demonstrateConnectionRecovery(server)

	// Demonstration 5: Resource Management
	fmt.Println("\n📈 DEMO 5: Resource Management")
	fmt.Println("------------------------------")
	demonstrateResourceManagement(server)

	fmt.Println("\n✅ Connection Pool Demonstration Complete!")
	fmt.Println("Benefits demonstrated:")
	fmt.Println("  • Connection reuse reduces latency")
	fmt.Println("  • Health monitoring ensures reliability")
	fmt.Println("  • Automatic recovery handles failures")
	fmt.Println("  • Concurrent operations are efficient")
	fmt.Println("  • Resource cleanup prevents leaks")
}

// demonstratePerformance shows the performance benefits of connection pooling
func demonstratePerformance(server *models.Server) {
	const numOperations = 10

	fmt.Println("Testing connection creation time...")

	// Test 1: Direct SSH managers (no pooling)
	fmt.Println("1. Direct SSH Connections (no pooling):")
	start := time.Now()
	for i := 0; i < numOperations; i++ {
		manager, err := ssh.NewSSHManager(server, false)
		if err != nil {
			fmt.Printf("   ❌ Connection %d failed: %v\n", i+1, err)
			continue
		}

		_, err = manager.ExecuteCommand("echo 'test'")
		if err != nil {
			fmt.Printf("   ❌ Command %d failed: %v\n", i+1, err)
		} else {
			fmt.Printf("   ✅ Operation %d completed\n", i+1)
		}

		manager.Close()
	}
	directTime := time.Since(start)

	// Test 2: Connection pool
	fmt.Println("2. Connection Pool:")
	sshService := ssh.GetSSHService()
	start = time.Now()
	for i := 0; i < numOperations; i++ {
		_, err := sshService.ExecuteCommand(server, false, "echo 'test'")
		if err != nil {
			fmt.Printf("   ❌ Operation %d failed: %v\n", i+1, err)
		} else {
			fmt.Printf("   ✅ Operation %d completed\n", i+1)
		}
	}
	poolTime := time.Since(start)

	// Results
	fmt.Printf("\n📊 Performance Results:\n")
	fmt.Printf("   Direct connections: %v (avg: %v per operation)\n",
		directTime, directTime/numOperations)
	fmt.Printf("   Connection pool:    %v (avg: %v per operation)\n",
		poolTime, poolTime/numOperations)

	if poolTime < directTime {
		improvement := float64(directTime-poolTime) / float64(directTime) * 100
		fmt.Printf("   🚀 Pool is %.1f%% faster!\n", improvement)
	}
}

// demonstrateHealthMonitoring shows the health monitoring capabilities
func demonstrateHealthMonitoring(server *models.Server) {
	sshService := ssh.GetSSHService()

	// Establish a connection
	fmt.Println("Establishing connection for health monitoring...")
	_, err := sshService.ExecuteCommand(server, false, "echo 'health test'")
	if err != nil {
		fmt.Printf("❌ Failed to establish connection: %v\n", err)
		return
	}

	// Show initial health status
	fmt.Println("📊 Initial Health Status:")
	showHealthStatus(sshService, server)

	// Perform some operations to generate metrics
	fmt.Println("\nPerforming operations to generate health metrics...")
	commands := []string{
		"echo 'operation 1'",
		"ls /tmp",
		"date",
		"whoami",
		"uptime",
	}

	for _, cmd := range commands {
		fmt.Printf("   Executing: %s\n", cmd)
		_, err := sshService.ExecuteCommand(server, false, cmd)
		if err != nil {
			fmt.Printf("   ❌ Command failed: %v\n", err)
		} else {
			fmt.Printf("   ✅ Command succeeded\n")
		}
		time.Sleep(100 * time.Millisecond) // Small delay between commands
	}

	// Show updated health status
	fmt.Println("\n📊 Updated Health Status:")
	showHealthStatus(sshService, server)
}

// demonstrateConcurrentOperations shows how the pool handles concurrent requests
func demonstrateConcurrentOperations(server *models.Server) {
	const numGoroutines = 5
	const operationsPerGoroutine = 3

	sshService := ssh.GetSSHService()
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make(map[int][]string)

	fmt.Printf("Running %d concurrent operations...\n", numGoroutines*operationsPerGoroutine)

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			var goroutineResults []string
			for j := 0; j < operationsPerGoroutine; j++ {
				cmd := fmt.Sprintf("echo 'Worker %d - Operation %d - Time: %s'",
					goroutineID, j+1, time.Now().Format("15:04:05.000"))

				output, err := sshService.ExecuteCommand(server, false, cmd)
				if err != nil {
					goroutineResults = append(goroutineResults,
						fmt.Sprintf("❌ Error: %v", err))
				} else {
					goroutineResults = append(goroutineResults,
						fmt.Sprintf("✅ %s", output))
				}
			}

			mu.Lock()
			results[goroutineID] = goroutineResults
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	// Show results
	fmt.Printf("\n📊 Concurrent Operations Results (completed in %v):\n", duration)
	for i := 0; i < numGoroutines; i++ {
		fmt.Printf("   Worker %d:\n", i)
		for _, result := range results[i] {
			fmt.Printf("     %s\n", result)
		}
	}

	// Show final connection status
	fmt.Println("\n📊 Connection Pool Status After Concurrent Operations:")
	showHealthStatus(sshService, server)
}

// demonstrateConnectionRecovery shows automatic connection recovery
func demonstrateConnectionRecovery(server *models.Server) {
	sshService := ssh.GetSSHService()

	// Establish initial connection
	fmt.Println("Establishing initial connection...")
	_, err := sshService.ExecuteCommand(server, false, "echo 'initial connection'")
	if err != nil {
		fmt.Printf("❌ Failed to establish connection: %v\n", err)
		return
	}
	fmt.Println("✅ Initial connection established")

	// Show connection health
	fmt.Println("\n📊 Initial Connection Health:")
	key := sshService.GetConnectionKey(server, false)
	isHealthy := sshService.IsConnectionHealthy(server, false)
	fmt.Printf("   Connection Key: %s\n", key)
	fmt.Printf("   Healthy: %v\n", isHealthy)

	// Simulate connection issues by attempting recovery
	fmt.Println("\n🔄 Testing Connection Recovery:")
	fmt.Println("   Attempting to recover connection...")

	err = sshService.RecoverConnection(server, false)
	if err != nil {
		fmt.Printf("   ⚠️  Recovery attempt: %v\n", err)
	} else {
		fmt.Printf("   ✅ Connection recovery successful\n")
	}

	// Test connection after recovery attempt
	fmt.Println("   Testing connection after recovery...")
	_, err = sshService.ExecuteCommand(server, false, "echo 'post-recovery test'")
	if err != nil {
		fmt.Printf("   ❌ Post-recovery test failed: %v\n", err)
	} else {
		fmt.Printf("   ✅ Post-recovery test successful\n")
	}
}

// demonstrateResourceManagement shows connection cleanup and resource management
func demonstrateResourceManagement(server *models.Server) {
	sshService := ssh.GetSSHService()

	// Show initial metrics
	fmt.Println("📊 Initial Resource Status:")
	metrics := sshService.GetHealthMetrics()
	fmt.Printf("   Total Connections: %d\n", metrics.TotalConnections)
	fmt.Printf("   Healthy Connections: %d\n", metrics.HealthyConnections)
	fmt.Printf("   Unhealthy Connections: %d\n", metrics.UnhealthyConnections)

	// Create some connections for different users
	fmt.Println("\nCreating connections for resource monitoring...")

	// App user connection
	_, err := sshService.ExecuteCommand(server, false, "echo 'app user test'")
	if err != nil {
		fmt.Printf("❌ App user connection failed: %v\n", err)
	} else {
		fmt.Printf("✅ App user connection established\n")
	}

	// Root user connection (if not security locked)
	if !server.SecurityLocked {
		_, err = sshService.ExecuteCommand(server, true, "echo 'root user test'")
		if err != nil {
			fmt.Printf("❌ Root user connection failed: %v\n", err)
		} else {
			fmt.Printf("✅ Root user connection established\n")
		}
	}

	// Show updated metrics
	fmt.Println("\n📊 Updated Resource Status:")
	metrics = sshService.GetHealthMetrics()
	fmt.Printf("   Total Connections: %d\n", metrics.TotalConnections)
	fmt.Printf("   Healthy Connections: %d\n", metrics.HealthyConnections)
	fmt.Printf("   Unhealthy Connections: %d\n", metrics.UnhealthyConnections)
	fmt.Printf("   Average Response Time: %v\n", metrics.AverageResponseTime)
	fmt.Printf("   Error Rate: %.2f%%\n", metrics.ErrorRate*100)

	// Demonstrate cleanup
	fmt.Println("\n🧹 Testing Connection Cleanup:")
	cleaned := sshService.CleanupConnections()
	fmt.Printf("   Cleaned up %d stale connections\n", cleaned)

	// Show final status
	connectionStatus := sshService.GetConnectionStatus()
	fmt.Printf("   Active connections in pool: %d\n", len(connectionStatus))

	for key, status := range connectionStatus {
		fmt.Printf("     %s: healthy=%v, age=%v, use_count=%d\n",
			key, status.Healthy, status.Age, status.UseCount)
	}
}

// showHealthStatus displays current health status
func showHealthStatus(sshService *ssh.SSHService, server *models.Server) {
	metrics := sshService.GetHealthMetrics()
	fmt.Printf("   Total Connections: %d\n", metrics.TotalConnections)
	fmt.Printf("   Healthy Connections: %d\n", metrics.HealthyConnections)
	fmt.Printf("   Unhealthy Connections: %d\n", metrics.UnhealthyConnections)
	fmt.Printf("   Average Response Time: %v\n", metrics.AverageResponseTime)
	fmt.Printf("   Error Rate: %.2f%%\n", metrics.ErrorRate*100)
	fmt.Printf("   Last Update: %v\n", metrics.LastUpdate.Format("15:04:05"))

	// Show specific connection health
	appHealthy := sshService.IsConnectionHealthy(server, false)
	rootHealthy := sshService.IsConnectionHealthy(server, true)
	fmt.Printf("   App User Connection: %v\n", appHealthy)

	if !server.SecurityLocked {
		fmt.Printf("   Root User Connection: %v\n", rootHealthy)
	} else {
		fmt.Printf("   Root User Connection: disabled (security locked)\n")
	}
}
