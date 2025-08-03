package cmd

import (
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

// healthCmd represents the health command for container health checks
var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check service health status",
	Long: `Check the health status of the SimpleIdentity service.

This command is designed for container health checks and Kubernetes probes.
It performs a quick HTTP request to the health endpoint and returns
appropriate exit codes for container orchestration.

Exit Codes:
  0 - Service is healthy
  1 - Service is unhealthy or unreachable`,
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, _ := cmd.Flags().GetString("addr")
		timeout, _ := cmd.Flags().GetDuration("timeout")

		return checkHealth(addr, timeout)
	},
}

func init() {
	rootCmd.AddCommand(healthCmd)

	healthCmd.Flags().String("addr", "localhost:8080", "Health check server address")
	healthCmd.Flags().Duration("timeout", 5*time.Second, "Request timeout")
}

func checkHealth(addr string, timeout time.Duration) error {
	client := &http.Client{
		Timeout: timeout,
	}

	url := fmt.Sprintf("http://%s/health", addr)

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: status %d", resp.StatusCode)
	}

	fmt.Println("Service is healthy")
	return nil
}
