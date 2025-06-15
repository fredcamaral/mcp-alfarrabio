package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"lerian-mcp-memory-cli/internal/domain/constants"
	"lerian-mcp-memory-cli/internal/domain/entities"
)

// createSetupCommand creates the setup wizard command
func (c *CLI) createSetupCommand() *cobra.Command {
	var (
		provider    string
		testAll     bool
		interactive bool
	)

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Interactive configuration wizard",
		Long: `Interactive configuration wizard to set up and test all connections.
		
This wizard helps you:
- Configure AI providers (OpenAI, Anthropic, etc.)
- Set up MCP server connection
- Configure default settings
- Test all connections
- Create initial project structure`,
		Example: `  # Run interactive setup
  lmmc setup
  
  # Test all connections
  lmmc setup --test
  
  # Configure specific provider
  lmmc setup --provider anthropic`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if testAll {
				return c.runSetupTest(cmd)
			}
			if provider != "" {
				return c.runSetupProvider(cmd, provider)
			}
			return c.runInteractiveSetup(cmd)
		},
	}

	cmd.Flags().StringVarP(&provider, "provider", "p", "", "Configure specific AI provider")
	cmd.Flags().BoolVar(&testAll, "test", false, "Test all configured connections")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", true, "Run in interactive mode")

	return cmd
}

// runInteractiveSetup runs the interactive configuration wizard
func (c *CLI) runInteractiveSetup(cmd *cobra.Command) error {
	reader := bufio.NewReader(os.Stdin)
	out := cmd.OutOrStdout()

	fmt.Fprintf(out, "🚀 Welcome to Lerian MCP Memory CLI Setup Wizard\n")
	fmt.Fprintf(out, "================================================\n\n")

	config, err := c.loadOrCreateConfig(out, reader)
	if err != nil {
		return err
	}

	if err := c.configureBasicSettings(out, reader, config); err != nil {
		return err
	}

	if err := c.configureMCPServer(out, reader, config); err != nil {
		return err
	}

	availableProviders := c.configureAIProviders(out, config)

	if err := c.setupProjectStructure(out, reader); err != nil {
		return err
	}

	if err := c.saveAndSummarizeConfig(out, config, availableProviders); err != nil {
		return err
	}

	return nil
}

// loadOrCreateConfig loads existing config or creates a new one
func (c *CLI) loadOrCreateConfig(out io.Writer, reader *bufio.Reader) (*entities.Config, error) {
	configFile := filepath.Join(os.Getenv("HOME"), ".lmmc", "config.yaml")

	if _, err := os.Stat(configFile); err == nil {
		fmt.Fprintf(out, "📋 Existing configuration found at: %s\n", configFile)
		fmt.Fprintf(out, "Would you like to update it? [Y/n]: ")

		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "n" || response == "no" {
			fmt.Fprintf(out, "Setup cancelled.\n")
			return nil, errors.New("setup cancelled by user")
		}

		// Load existing config
		if config, err := c.configMgr.Load(); err == nil {
			return config, nil
		}
	}

	return &entities.Config{
		CLI: entities.CLIConfig{
			DefaultRepository: "default",
			OutputFormat:      "table",
			PageSize:          20,
		},
		Server: entities.ServerConfig{
			URL:     "http://localhost:9080",
			Timeout: 30,
		},
	}, nil
}

// configureBasicSettings handles Step 1: Basic Configuration
func (c *CLI) configureBasicSettings(out io.Writer, reader *bufio.Reader, config *entities.Config) error {
	fmt.Fprintf(out, "\n📝 Step 1: Basic Configuration\n")
	fmt.Fprintf(out, "------------------------------\n")

	// Default repository
	fmt.Fprintf(out, "\nDefault repository name [%s]: ", config.CLI.DefaultRepository)
	if input, _ := reader.ReadString('\n'); strings.TrimSpace(input) != "" {
		config.CLI.DefaultRepository = strings.TrimSpace(input)
	}

	// Output format
	fmt.Fprintf(out, "Default output format (table/json/plain) [%s]: ", config.CLI.OutputFormat)
	if input, _ := reader.ReadString('\n'); strings.TrimSpace(input) != "" {
		config.CLI.OutputFormat = strings.TrimSpace(input)
	}

	return nil
}

// configureMCPServer handles Step 2: MCP Server Configuration
func (c *CLI) configureMCPServer(out io.Writer, reader *bufio.Reader, config *entities.Config) error {
	fmt.Fprintf(out, "\n🔌 Step 2: MCP Server Configuration\n")
	fmt.Fprintf(out, "-----------------------------------\n")

	fmt.Fprintf(out, "\nMCP Server URL [%s]: ", config.Server.URL)
	if input, _ := reader.ReadString('\n'); strings.TrimSpace(input) != "" {
		config.Server.URL = strings.TrimSpace(input)
	}

	fmt.Fprintf(out, "Enable auto-sync? (y/n) [y]: ")
	_, _ = reader.ReadString('\n') // Just read and ignore for now

	// Test MCP connection
	fmt.Fprintf(out, "\n🔍 Testing MCP server connection...\n")
	if err := c.testMCPConnection(config.Server.URL); err != nil {
		fmt.Fprintf(out, "❌ MCP server connection failed: %v\n", err)
		fmt.Fprintf(out, "   Make sure the server is running: docker-compose --profile dev up\n")
	} else {
		fmt.Fprintf(out, "✅ MCP server connection successful!\n")
	}

	return nil
}

// configureAIProviders handles Step 3: AI Provider Configuration
func (c *CLI) configureAIProviders(out io.Writer, config *entities.Config) []string {
	_ = config // Currently unused, but kept for future extension
	fmt.Fprintf(out, "\n🤖 Step 3: AI Provider Configuration\n")
	fmt.Fprintf(out, "------------------------------------\n")

	providers := []struct {
		name   string
		envVar string
		desc   string
	}{
		{"openai", "OPENAI_API_KEY", "OpenAI (GPT-4, GPT-3.5)"},
		{"anthropic", "ANTHROPIC_API_KEY", "Anthropic (Claude)"},
		{"perplexity", "PERPLEXITY_API_KEY", "Perplexity AI"},
	}

	availableProviders := []string{}
	for _, p := range providers {
		if os.Getenv(p.envVar) != "" {
			availableProviders = append(availableProviders, p.name)
			fmt.Fprintf(out, "✅ %s API key found in environment\n", p.desc)
		} else {
			fmt.Fprintf(out, "❌ %s API key not found (%s)\n", p.desc, p.envVar)
		}
	}

	if len(availableProviders) == 0 {
		fmt.Fprintf(out, "\n⚠️  No AI provider API keys found in environment.\n")
		fmt.Fprintf(out, "   Set environment variables before using AI features:\n")
		for _, p := range providers {
			fmt.Fprintf(out, "   export %s=your-api-key\n", p.envVar)
		}
	}

	return availableProviders
}

// setupProjectStructure handles Step 4: Project Structure
func (c *CLI) setupProjectStructure(out io.Writer, reader *bufio.Reader) error {
	fmt.Fprintf(out, "\n📁 Step 4: Project Structure\n")
	fmt.Fprintf(out, "----------------------------\n")

	fmt.Fprintf(out, "\nWould you like to create standard project directories? [Y/n]: ")
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "n" && response != "no" {
		dirs := []string{
			constants.DefaultPreDevelopmentDir,
			"docs/tasks",
			"docs/reviews",
			".lmmc",
		}

		for _, dir := range dirs {
			if err := os.MkdirAll(dir, 0750); err != nil {
				fmt.Fprintf(out, "❌ Failed to create %s: %v\n", dir, err)
			} else {
				fmt.Fprintf(out, "✅ Created %s\n", dir)
			}
		}
	}

	return nil
}

// saveAndSummarizeConfig handles Step 5: Save Configuration and final summary
func (c *CLI) saveAndSummarizeConfig(out io.Writer, config *entities.Config, availableProviders []string) error {
	fmt.Fprintf(out, "\n💾 Step 5: Save Configuration\n")
	fmt.Fprintf(out, "-----------------------------\n")

	configDir := filepath.Join(os.Getenv("HOME"), ".lmmc")
	configFile := filepath.Join(configDir, "config.yaml")

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Save configuration
	if err := c.saveConfig(config, configFile); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Fprintf(out, "✅ Configuration saved to: %s\n", configFile)

	// Final summary
	fmt.Fprintf(out, "\n✨ Setup Complete!\n")
	fmt.Fprintf(out, "==================\n\n")
	fmt.Fprintf(out, "Your configuration:\n")
	fmt.Fprintf(out, "- Default repository: %s\n", config.CLI.DefaultRepository)
	fmt.Fprintf(out, "- Output format: %s\n", config.CLI.OutputFormat)
	fmt.Fprintf(out, "- MCP Server: %s\n", config.Server.URL)
	if len(availableProviders) > 0 {
		fmt.Fprintf(out, "- AI Providers available: %s\n", strings.Join(availableProviders, ", "))
	}

	fmt.Fprintf(out, "\nNext steps:\n")
	fmt.Fprintf(out, "1. Test your setup: lmmc setup --test\n")
	fmt.Fprintf(out, "2. Create your first PRD: lmmc prd create \"Your feature\"\n")
	fmt.Fprintf(out, "3. Generate sample data: lmmc generate sample-tasks\n")
	fmt.Fprintf(out, "\nHappy coding! 🚀\n")

	return nil
}

// runSetupTest tests all configured connections
func (c *CLI) runSetupTest(cmd *cobra.Command) error {
	fmt.Fprintf(cmd.OutOrStdout(), "🔍 Testing all connections...\n")
	fmt.Fprintf(cmd.OutOrStdout(), "============================\n\n")

	// Load configuration
	config, err := c.configMgr.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	allPassed := true

	// Test 1: Configuration
	fmt.Fprintf(cmd.OutOrStdout(), "1. Configuration\n")
	fmt.Fprintf(cmd.OutOrStdout(), "   ✅ Config loaded from: %s\n", viper.ConfigFileUsed())
	fmt.Fprintf(cmd.OutOrStdout(), "   ✅ Default repository: %s\n", config.CLI.DefaultRepository)

	// Test 2: File System
	fmt.Fprintf(cmd.OutOrStdout(), "\n2. File System\n")
	testDirs := []string{".lmmc", "docs"}
	for _, dir := range testDirs {
		if _, err := os.Stat(dir); err == nil {
			fmt.Fprintf(cmd.OutOrStdout(), "   ✅ Directory exists: %s\n", dir)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "   ❌ Directory missing: %s\n", dir)
			allPassed = false
		}
	}

	// Test 3: MCP Server
	fmt.Fprintf(cmd.OutOrStdout(), "\n3. MCP Server Connection\n")
	if err := c.testMCPConnection(config.Server.URL); err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "   ❌ Connection failed: %v\n", err)
		fmt.Fprintf(cmd.OutOrStdout(), "   💡 Start server: docker-compose --profile dev up\n")
		allPassed = false
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "   ✅ Server is reachable at: %s\n", config.Server.URL)

		// Test MCP tools
		if mcpClient := c.getMCPClient(); mcpClient != nil {
			ctx, cancel := context.WithTimeout(c.getContext(), 5*time.Second)
			defer cancel()

			if result, err := mcpClient.CallMCPTool(ctx, "tools/list", nil); err == nil {
				if tools, ok := result["tools"].([]interface{}); ok {
					fmt.Fprintf(cmd.OutOrStdout(), "   ✅ Found %d MCP tools\n", len(tools))
				}
			}
		}
	}

	// Test 4: AI Provider
	fmt.Fprintf(cmd.OutOrStdout(), "\n4. AI Provider\n")
	if c.aiService != nil {
		ctx, cancel := context.WithTimeout(c.getContext(), 5*time.Second)
		defer cancel()

		if err := c.aiService.TestConnection(ctx); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "   ❌ AI provider test failed: %v\n", err)
			allPassed = false
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "   ✅ AI provider is working\n")
		}
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "   ⚠️  No AI provider configured\n")
	}

	// Test 5: Local Storage
	fmt.Fprintf(cmd.OutOrStdout(), "\n5. Local Storage\n")
	if c.storage != nil {
		// Try to list tasks
		ctx := c.getContext()
		if _, err := c.storage.ListTasks(ctx, "", nil); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "   ❌ Storage access failed: %v\n", err)
			allPassed = false
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "   ✅ Local task storage is working\n")
		}
	}

	// Summary
	fmt.Fprintf(cmd.OutOrStdout(), "\n=============================\n")
	if allPassed {
		fmt.Fprintf(cmd.OutOrStdout(), "✅ All tests passed!\n")
		fmt.Fprintf(cmd.OutOrStdout(), "\nYour CLI is ready to use. Try:\n")
		fmt.Fprintf(cmd.OutOrStdout(), "- lmmc add \"Your first task\"\n")
		fmt.Fprintf(cmd.OutOrStdout(), "- lmmc generate sample-tasks\n")
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "❌ Some tests failed.\n")
		fmt.Fprintf(cmd.OutOrStdout(), "\nFix the issues above and run 'lmmc setup --test' again.\n")
	}

	return nil
}

// runSetupProvider configures a specific AI provider
func (c *CLI) runSetupProvider(cmd *cobra.Command, provider string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Fprintf(cmd.OutOrStdout(), "🤖 Configuring AI Provider: %s\n", provider)
	fmt.Fprintf(cmd.OutOrStdout(), "================================\n\n")

	// Load existing config
	config, err := c.configMgr.Load()
	if err != nil {
		config = entities.DefaultConfig()
	}

	// Provider-specific configuration
	switch provider {
	case "anthropic":
		fmt.Fprintf(cmd.OutOrStdout(), "Anthropic (Claude) Configuration\n")
		fmt.Fprintf(cmd.OutOrStdout(), "---------------------------------\n\n")

		// Check for API key
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			fmt.Fprintf(cmd.OutOrStdout(), "❌ ANTHROPIC_API_KEY not found in environment\n")
			fmt.Fprintf(cmd.OutOrStdout(), "\nTo set it:\n")
			fmt.Fprintf(cmd.OutOrStdout(), "export ANTHROPIC_API_KEY=your-api-key\n\n")
			return errors.New("ANTHROPIC_API_KEY not set")
		}

		fmt.Fprintf(cmd.OutOrStdout(), "✅ API key found (starts with: %s...)\n", apiKey[:10])

		// Configure default model
		fmt.Fprintf(cmd.OutOrStdout(), "\nDefault model (claude-3-opus-20240229, claude-3-sonnet-20240229) [claude-3-opus-20240229]: ")
		model, _ := reader.ReadString('\n')
		model = strings.TrimSpace(model)
		if model == "" {
			model = "claude-3-opus-20240229"
		}

		// For now, just store the model choice in a comment
		fmt.Fprintf(cmd.OutOrStdout(), "\n✅ Anthropic model preference noted: %s\n", model)

	case "openai":
		fmt.Fprintf(cmd.OutOrStdout(), "OpenAI Configuration\n")
		fmt.Fprintf(cmd.OutOrStdout(), "--------------------\n\n")

		// Check for API key
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			fmt.Fprintf(cmd.OutOrStdout(), "❌ OPENAI_API_KEY not found in environment\n")
			fmt.Fprintf(cmd.OutOrStdout(), "\nTo set it:\n")
			fmt.Fprintf(cmd.OutOrStdout(), "export OPENAI_API_KEY=your-api-key\n\n")
			return errors.New("OPENAI_API_KEY not set")
		}

		fmt.Fprintf(cmd.OutOrStdout(), "✅ API key found (starts with: %s...)\n", apiKey[:10])

		// Configure default model
		fmt.Fprintf(cmd.OutOrStdout(), "\nDefault model (gpt-4, gpt-4-turbo, gpt-3.5-turbo) [gpt-4]: ")
		model, _ := reader.ReadString('\n')
		model = strings.TrimSpace(model)
		if model == "" {
			model = "gpt-4"
		}

		// For now, just store the model choice in a comment
		fmt.Fprintf(cmd.OutOrStdout(), "\n✅ OpenAI model preference noted: %s\n", model)

	default:
		return fmt.Errorf("unknown provider: %s", provider)
	}

	// Save configuration
	configDir := filepath.Join(os.Getenv("HOME"), ".lmmc")
	configFile := filepath.Join(configDir, "config.yaml")

	if err := os.MkdirAll(configDir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := c.saveConfig(config, configFile); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n✅ Provider configured successfully!\n")
	fmt.Fprintf(cmd.OutOrStdout(), "   Provider: %s\n", provider)

	// Test the provider
	fmt.Fprintf(cmd.OutOrStdout(), "\n🔍 Testing provider connection...\n")
	if c.aiService != nil {
		ctx, cancel := context.WithTimeout(c.getContext(), 10*time.Second)
		defer cancel()

		if err := c.aiService.TestConnection(ctx); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "❌ Connection test failed: %v\n", err)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "✅ Provider is working!\n")
		}
	}

	return nil
}

// Helper functions

func (c *CLI) testMCPConnection(url string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to get MCP client and test connection
	if mcpClient := c.getMCPClient(); mcpClient != nil {
		return mcpClient.TestConnection(ctx)
	}

	// Fallback to basic HTTP check
	healthURL := strings.TrimSuffix(url, "/") + "/health"
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			// Log error but don't return as we're in defer
			_ = err // Explicitly acknowledge we're discarding the error
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *CLI) saveConfig(config *entities.Config, path string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}
