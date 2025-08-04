package cmd

import (
	"context"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/posilva/simpleidentity/pkg/config"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "simpleidentity",
	Short: "Enterprise-grade identity management service for gaming platforms",
	Long: `SimpleIdentity is a secure, scalable identity management service designed for modern gaming platforms.

Built with enterprise-grade security and performance in mind, SimpleIdentity provides seamless authentication
and authorization services that support multiple identity providers including:

• Guest authentication for anonymous gameplay
• Apple Sign-In integration for iOS users  
• Google OAuth for cross-platform access
• Extensible provider architecture for future integrations


Environment Variables:
  All command-line flags can be set via environment variables with SMPIDT_ prefix.
  Examples: SMPIDT_PORT=8080, SMPIDT_DB_URL=dynamodb://local`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// ExecuteContext adds all child commands to the root command and sets flags appropriately.
// This version accepts a context for cancellation and timeout control.
func ExecuteContext(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	cobra.OnInitialize(initConfig)
}

// initConfig sets up environment variable binding for 12-factor app compliance
func initConfig() {
	// Initialize global configuration manager
	configMgr := config.InitGlobal()

	// Enable automatic environment variable binding
	viper.AutomaticEnv()
	viper.SetEnvPrefix("SMPIDT")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.BindPFlags(rootCmd.PersistentFlags())

	// Set defaults through the config manager
	configMgr.SetDefaults()
}
