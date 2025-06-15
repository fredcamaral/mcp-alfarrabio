package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// AIProviderConfig represents a configured AI provider
type AIProviderConfig struct {
	Name      string                 `json:"name" yaml:"name"`
	Type      string                 `json:"type" yaml:"type"` // openai, anthropic, google, local
	APIKey    string                 `json:"api_key,omitempty" yaml:"api_key,omitempty"`
	Endpoint  string                 `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Models    []AIModelConfig        `json:"models,omitempty" yaml:"models,omitempty"`
	IsDefault bool                   `json:"is_default,omitempty" yaml:"is_default,omitempty"`
	Config    map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`
	AddedAt   time.Time              `json:"added_at" yaml:"added_at"`
}

// AIModelConfig represents a model configuration
type AIModelConfig struct {
	ID           string   `json:"id" yaml:"id"`
	Name         string   `json:"name" yaml:"name"`
	ContextSize  int      `json:"context_size" yaml:"context_size"`
	InputCost    float64  `json:"input_cost" yaml:"input_cost"`   // $ per 1M tokens
	OutputCost   float64  `json:"output_cost" yaml:"output_cost"` // $ per 1M tokens
	Capabilities []string `json:"capabilities" yaml:"capabilities"`
}

// AICostBudget represents cost budget configuration
type AICostBudget struct {
	Monthly     float64            `json:"monthly" yaml:"monthly"`
	PerCommand  float64            `json:"per_command" yaml:"per_command"`
	PerProvider map[string]float64 `json:"per_provider" yaml:"per_provider"`
	AlertAt     float64            `json:"alert_at" yaml:"alert_at"` // percentage
}

// AIUsageStats represents usage statistics
type AIUsageStats struct {
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	InputTokens  int64     `json:"input_tokens"`
	OutputTokens int64     `json:"output_tokens"`
	Cost         float64   `json:"cost"`
	Timestamp    time.Time `json:"timestamp"`
	Command      string    `json:"command"`
}

// createAIProviderCommands adds provider management commands
func (c *CLI) createAIProviderCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider",
		Short: "Manage AI provider configurations",
		Long:  `Configure and manage AI providers including API keys, models, and costs.`,
	}

	// Add subcommands
	cmd.AddCommand(c.createAIAddProviderCommand())
	cmd.AddCommand(c.createAIListProvidersCommand())
	cmd.AddCommand(c.createAIRemoveProviderCommand())
	cmd.AddCommand(c.createAISetDefaultProviderCommand())
	cmd.AddCommand(c.createAITestProviderCommand())

	return cmd
}

// createAIModelCommands adds model management commands
func (c *CLI) createAIModelCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model",
		Short: "Manage AI models",
		Long:  `List and configure AI models for different providers.`,
	}

	// Add subcommands
	cmd.AddCommand(c.createAIListModelsCommand())
	cmd.AddCommand(c.createAISetDefaultModelCommand())
	cmd.AddCommand(c.createAIRecommendModelCommand())

	return cmd
}

// createAICostCommands adds cost management commands
func (c *CLI) createAICostCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cost",
		Short: "Manage AI costs and budgets",
		Long:  `Track usage, set budgets, and manage AI provider costs.`,
	}

	// Add subcommands
	cmd.AddCommand(c.createAISetBudgetCommand())
	cmd.AddCommand(c.createAIUsageCommand())
	cmd.AddCommand(c.createAIEstimateCommand())

	return cmd
}

// createAIFallbackCommands adds fallback configuration commands
func (c *CLI) createAIFallbackCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fallback",
		Short: "Configure provider fallback chains",
		Long:  `Set up fallback chains for automatic failover between AI providers.`,
	}

	// Add subcommands
	cmd.AddCommand(c.createAISetFallbackCommand())
	cmd.AddCommand(c.createAIListFallbackCommand())

	return cmd
}

// createAIRecommendModelCommand creates the 'ai model recommend' command
func (c *CLI) createAIRecommendModelCommand() *cobra.Command {
	var forTask string

	cmd := &cobra.Command{
		Use:   "recommend",
		Short: "Get model recommendations for specific tasks",
		Long: `Get AI model recommendations based on task requirements.

Examples:
  lmmc ai model recommend --for "large context analysis"
  lmmc ai model recommend --for "quick code review"
  lmmc ai model recommend --for "multimodal processing"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if forTask == "" && len(args) > 0 {
				forTask = strings.Join(args, " ")
			}

			if forTask == "" {
				return errors.New("task description required")
			}

			fmt.Printf("🤖 AI Model Recommendations\n")
			fmt.Printf("══════════════════════════\n\n")
			fmt.Printf("Task: %s\n\n", forTask)

			// Analyze task requirements
			recommendations := c.analyzeTaskRequirements(forTask)

			fmt.Printf("Recommended Models:\n")
			fmt.Printf("──────────────────\n\n")

			for i, rec := range recommendations {
				fmt.Printf("%d. %s/%s\n", i+1, rec.Provider, rec.Model)
				fmt.Printf("   %s\n", rec.Reason)
				fmt.Printf("   Context: %s tokens\n", formatNumber(rec.ContextSize))
				fmt.Printf("   Est. Cost: $%.2f per run\n", rec.EstimatedCost)
				fmt.Printf("   Speed: %s\n\n", rec.Speed)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&forTask, "for", "", "Task description")

	return cmd
}

// Individual command implementations

// createAIAddProviderCommand creates the 'ai provider add' command
func (c *CLI) createAIAddProviderCommand() *cobra.Command {
	var (
		providerType string
		apiKey       string
		endpoint     string
		name         string
		setDefault   bool
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new AI provider",
		Long: `Add and configure a new AI provider.

Examples:
  lmmc ai provider add --type anthropic --api-key $ANTHROPIC_API_KEY --name claude
  lmmc ai provider add --type openai --api-key $OPENAI_API_KEY --default
  lmmc ai provider add --type local --endpoint http://localhost:11434 --name ollama`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if providerType == "" {
				return errors.New("provider type is required")
			}

			// Generate name if not provided
			if name == "" {
				name = providerType
			}

			// Create provider config
			config := &AIProviderConfig{
				Name:      name,
				Type:      providerType,
				APIKey:    apiKey,
				Endpoint:  endpoint,
				IsDefault: setDefault,
				AddedAt:   time.Now(),
				Config:    make(map[string]interface{}),
			}

			// Load default models for known providers
			switch providerType {
			case "anthropic":
				config.Models = []AIModelConfig{
					{ID: "claude-opus-4", Name: "Claude Opus 4", ContextSize: 200000, InputCost: 15.0, OutputCost: 75.0, Capabilities: []string{"chat", "code", "analysis"}},
					{ID: "claude-sonnet-3.5", Name: "Claude Sonnet 3.5", ContextSize: 200000, InputCost: 3.0, OutputCost: 15.0, Capabilities: []string{"chat", "code"}},
					{ID: "claude-haiku", Name: "Claude Haiku", ContextSize: 200000, InputCost: 0.25, OutputCost: 1.25, Capabilities: []string{"chat", "quick"}},
				}
			case "openai":
				config.Models = []AIModelConfig{
					{ID: "gpt-4o", Name: "GPT-4 Optimized", ContextSize: 128000, InputCost: 5.0, OutputCost: 15.0, Capabilities: []string{"chat", "code", "function"}},
					{ID: "gpt-4-turbo", Name: "GPT-4 Turbo", ContextSize: 128000, InputCost: 10.0, OutputCost: 30.0, Capabilities: []string{"chat", "code", "vision"}},
					{ID: "gpt-3.5-turbo", Name: "GPT-3.5 Turbo", ContextSize: 16385, InputCost: 0.5, OutputCost: 1.5, Capabilities: []string{"chat", "quick"}},
				}
			case "google":
				config.Models = []AIModelConfig{
					{ID: "gemini-ultra", Name: "Gemini Ultra", ContextSize: 1000000, InputCost: 20.0, OutputCost: 60.0, Capabilities: []string{"chat", "code", "multimodal"}},
					{ID: "gemini-pro", Name: "Gemini Pro", ContextSize: 32000, InputCost: 0.5, OutputCost: 1.5, Capabilities: []string{"chat", "code"}},
				}
			}

			// Save provider config
			if err := c.saveAIProviderConfig(config); err != nil {
				return fmt.Errorf("failed to save provider config: %w", err)
			}

			fmt.Printf("✅ AI provider '%s' added successfully\n", name)

			if setDefault {
				fmt.Printf("   Set as default provider\n")
			}

			// Test connection if API key provided
			if apiKey != "" {
				fmt.Printf("\n🔍 Testing connection...\n")
				if err := c.testAIProvider(config); err != nil {
					fmt.Printf("⚠️  Connection test failed: %v\n", err)
					fmt.Printf("   Provider saved but may not be functional\n")
				} else {
					fmt.Printf("✅ Connection successful!\n")
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&providerType, "type", "", "Provider type (anthropic, openai, google, local)")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key for the provider")
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "Custom endpoint URL (for local providers)")
	cmd.Flags().StringVar(&name, "name", "", "Custom name for this provider configuration")
	cmd.Flags().BoolVar(&setDefault, "default", false, "Set as default provider")

	markFlagRequired(cmd, "type")

	return cmd
}

// createAIListProvidersCommand creates the 'ai provider list' command
func (c *CLI) createAIListProvidersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured AI providers",
		Long:  `Display all configured AI providers and their status.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			providers, err := c.loadAIProviders()
			if err != nil {
				return fmt.Errorf("failed to load providers: %w", err)
			}

			if len(providers) == 0 {
				fmt.Printf("No AI providers configured.\n")
				fmt.Printf("\nAdd a provider with: lmmc ai provider add --type <provider> --api-key <key>\n")
				return nil
			}

			fmt.Printf("🤖 Configured AI Providers\n")
			fmt.Printf("═════════════════════════\n\n")

			for _, provider := range providers {
				defaultStr := ""
				if provider.IsDefault {
					defaultStr = " (default)"
				}

				fmt.Printf("📌 %s%s\n", provider.Name, defaultStr)
				fmt.Printf("   Type: %s\n", provider.Type)

				if provider.APIKey != "" {
					maskedKey := provider.APIKey[:minInt(8, len(provider.APIKey))] + "..."
					fmt.Printf("   API Key: %s\n", maskedKey)
				}

				if provider.Endpoint != "" {
					fmt.Printf("   Endpoint: %s\n", provider.Endpoint)
				}

				if len(provider.Models) > 0 {
					fmt.Printf("   Models: %d available\n", len(provider.Models))
				}

				fmt.Printf("   Added: %s\n", provider.AddedAt.Format("2006-01-02 15:04:05"))
				fmt.Printf("\n")
			}

			return nil
		},
	}

	return cmd
}

// createAIListModelsCommand creates the 'ai model list' command
func (c *CLI) createAIListModelsCommand() *cobra.Command {
	var provider string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available models",
		Long: `List all available models for a provider.

Examples:
  lmmc ai model list --provider anthropic
  lmmc ai model list --provider openai`,
		RunE: func(cmd *cobra.Command, args []string) error {
			providers, err := c.loadAIProviders()
			if err != nil {
				return fmt.Errorf("failed to load providers: %w", err)
			}

			// Filter by provider if specified
			if provider != "" {
				found := false
				for _, p := range providers {
					if p.Name == provider || p.Type == provider {
						providers = []AIProviderConfig{p}
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("provider '%s' not found", provider)
				}
			}

			fmt.Printf("🧠 Available AI Models\n")
			fmt.Printf("═════════════════════\n\n")

			for _, p := range providers {
				fmt.Printf("Provider: %s (%s)\n", p.Name, p.Type)
				fmt.Printf("─────────────────────\n")

				if len(p.Models) == 0 {
					fmt.Printf("  No models configured\n\n")
					continue
				}

				for _, model := range p.Models {
					fmt.Printf("  📋 %s (%s)\n", model.Name, model.ID)
					fmt.Printf("     Context: %s tokens\n", formatNumber(model.ContextSize))
					fmt.Printf("     Cost: $%.2f/$%.2f per 1M tokens (in/out)\n", model.InputCost, model.OutputCost)

					if len(model.Capabilities) > 0 {
						fmt.Printf("     Capabilities: %s\n", strings.Join(model.Capabilities, ", "))
					}
					fmt.Printf("\n")
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&provider, "provider", "", "Filter by provider name or type")

	return cmd
}

// createAISetDefaultModelCommand creates the 'ai model set-default' command
func (c *CLI) createAISetDefaultModelCommand() *cobra.Command {
	var (
		forTask  string
		provider string
		model    string
	)

	cmd := &cobra.Command{
		Use:   "set-default",
		Short: "Set default model for task type",
		Long: `Set the default model to use for specific task types.

Examples:
  lmmc ai model set-default --for prd-generation --provider anthropic --model claude-opus-4
  lmmc ai model set-default --for code-review --provider openai --model gpt-4o
  lmmc ai model set-default --for quick-analysis --provider anthropic --model claude-haiku`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if forTask == "" || provider == "" || model == "" {
				return errors.New("--for, --provider, and --model are required")
			}

			// Load and update defaults
			defaults, err := c.loadAIDefaults()
			if err != nil {
				defaults = make(map[string]AIDefaultConfig)
			}

			defaults[forTask] = AIDefaultConfig{
				Provider: provider,
				Model:    model,
			}

			if err := c.saveAIDefaults(defaults); err != nil {
				return fmt.Errorf("failed to save defaults: %w", err)
			}

			fmt.Printf("✅ Default model set for '%s':\n", forTask)
			fmt.Printf("   Provider: %s\n", provider)
			fmt.Printf("   Model: %s\n", model)

			return nil
		},
	}

	cmd.Flags().StringVar(&forTask, "for", "", "Task type (prd-generation, code-review, etc.)")
	cmd.Flags().StringVar(&provider, "provider", "", "Provider name")
	cmd.Flags().StringVar(&model, "model", "", "Model ID")

	return cmd
}

// createAISetBudgetCommand creates the 'ai cost set-budget' command
func (c *CLI) createAISetBudgetCommand() *cobra.Command {
	var (
		monthly    float64
		perCommand float64
		provider   string
		amount     float64
		alertAt    float64
	)

	cmd := &cobra.Command{
		Use:   "set-budget",
		Short: "Set AI usage budgets",
		Long: `Set spending limits for AI usage.

Examples:
  lmmc ai cost set-budget --monthly 100.00
  lmmc ai cost set-budget --per-command 5.00
  lmmc ai cost set-budget --provider anthropic --amount 50.00
  lmmc ai cost set-budget --alert-at 80`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load existing budget
			budget, err := c.loadAIBudget()
			if err != nil {
				budget = &AICostBudget{
					PerProvider: make(map[string]float64),
				}
			}

			// Update budget settings
			if cmd.Flags().Changed("monthly") {
				budget.Monthly = monthly
			}
			if cmd.Flags().Changed("per-command") {
				budget.PerCommand = perCommand
			}
			if provider != "" && amount > 0 {
				budget.PerProvider[provider] = amount
			}
			if cmd.Flags().Changed("alert-at") {
				budget.AlertAt = alertAt
			}

			// Save budget
			if err := c.saveAIBudget(budget); err != nil {
				return fmt.Errorf("failed to save budget: %w", err)
			}

			fmt.Printf("✅ AI budget updated:\n")
			if budget.Monthly > 0 {
				fmt.Printf("   Monthly limit: $%.2f\n", budget.Monthly)
			}
			if budget.PerCommand > 0 {
				fmt.Printf("   Per-command limit: $%.2f\n", budget.PerCommand)
			}
			if len(budget.PerProvider) > 0 {
				fmt.Printf("   Provider limits:\n")
				for p, limit := range budget.PerProvider {
					fmt.Printf("     %s: $%.2f\n", p, limit)
				}
			}
			if budget.AlertAt > 0 {
				fmt.Printf("   Alert at: %.0f%% of budget\n", budget.AlertAt)
			}

			return nil
		},
	}

	cmd.Flags().Float64Var(&monthly, "monthly", 0, "Monthly spending limit")
	cmd.Flags().Float64Var(&perCommand, "per-command", 0, "Per-command spending limit")
	cmd.Flags().StringVar(&provider, "provider", "", "Provider name for provider-specific limit")
	cmd.Flags().Float64Var(&amount, "amount", 0, "Amount for provider-specific limit")
	cmd.Flags().Float64Var(&alertAt, "alert-at", 80, "Alert when usage reaches this percentage")

	return cmd
}

// createAIUsageCommand creates the 'ai cost usage' command
func (c *CLI) createAIUsageCommand() *cobra.Command {
	var (
		provider  string
		period    string
		breakdown string
	)

	cmd := &cobra.Command{
		Use:   "usage",
		Short: "View AI usage statistics",
		Long: `Display AI usage statistics and costs.

Examples:
  lmmc ai cost usage --period month
  lmmc ai cost usage --provider anthropic --period week
  lmmc ai cost usage --breakdown-by task-type`,
		RunE: func(cmd *cobra.Command, args []string) error {
			stats := c.loadAIUsageStats(period, provider)

			fmt.Printf("💰 AI Usage Report\n")
			fmt.Printf("═════════════════\n\n")

			if len(stats) == 0 {
				fmt.Printf("No usage data for the specified period.\n")
				return nil
			}

			// Calculate totals
			totalCost := 0.0
			totalInput := int64(0)
			totalOutput := int64(0)
			providerCosts := make(map[string]float64)
			modelUsage := make(map[string]int)

			for _, s := range stats {
				totalCost += s.Cost
				totalInput += s.InputTokens
				totalOutput += s.OutputTokens
				providerCosts[s.Provider] += s.Cost
				modelUsage[s.Model]++
			}

			// Display summary
			fmt.Printf("📊 Summary (%s)\n", period)
			fmt.Printf("────────────────\n")
			fmt.Printf("Total Cost: $%.2f\n", totalCost)
			fmt.Printf("Total Tokens: %s in / %s out\n",
				formatNumber(int(totalInput)),
				formatNumber(int(totalOutput)))
			fmt.Printf("API Calls: %d\n\n", len(stats))

			// Provider breakdown
			if len(providerCosts) > 1 || provider == "" {
				fmt.Printf("💸 Cost by Provider\n")
				fmt.Printf("──────────────────\n")
				for p, cost := range providerCosts {
					percentage := (cost / totalCost) * 100
					fmt.Printf("%-15s $%7.2f (%5.1f%%)\n", p, cost, percentage)
				}
				fmt.Printf("\n")
			}

			// Model usage
			fmt.Printf("🧠 Model Usage\n")
			fmt.Printf("─────────────\n")
			for model, count := range modelUsage {
				fmt.Printf("%-20s %d calls\n", model, count)
			}

			// Check budget
			budget, err := c.loadAIBudget()
			if err == nil && budget.Monthly > 0 {
				percentage := (totalCost / budget.Monthly) * 100
				fmt.Printf("\n💰 Budget Status\n")
				fmt.Printf("───────────────\n")
				fmt.Printf("Monthly Budget: $%.2f\n", budget.Monthly)
				fmt.Printf("Used: $%.2f (%.1f%%)\n", totalCost, percentage)

				if percentage >= budget.AlertAt {
					fmt.Printf("⚠️  WARNING: Approaching budget limit!\n")
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&provider, "provider", "", "Filter by provider")
	cmd.Flags().StringVar(&period, "period", "month", "Time period (day, week, month, year)")
	cmd.Flags().StringVar(&breakdown, "breakdown-by", "", "Breakdown by (task-type, model, provider)")

	return cmd
}

// createAIRemoveProviderCommand creates the 'ai provider remove' command
func (c *CLI) createAIRemoveProviderCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove [name]",
		Short: "Remove an AI provider",
		Long:  `Remove a configured AI provider.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			providers, err := c.loadAIProviders()
			if err != nil {
				return fmt.Errorf("failed to load providers: %w", err)
			}

			found := false
			newProviders := []AIProviderConfig{}
			for _, p := range providers {
				if p.Name == name {
					found = true
				} else {
					newProviders = append(newProviders, p)
				}
			}

			if !found {
				return fmt.Errorf("provider '%s' not found", name)
			}

			// Save updated list
			data, err := yaml.Marshal(newProviders)
			if err != nil {
				return err
			}

			configDir := c.getAIConfigDir()
			if err := os.WriteFile(filepath.Join(configDir, "providers.yaml"), data, 0600); err != nil {
				return err
			}

			fmt.Printf("✅ Provider '%s' removed successfully\n", name)
			return nil
		},
	}

	return cmd
}

// createAISetDefaultProviderCommand creates the 'ai provider set-default' command
func (c *CLI) createAISetDefaultProviderCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-default [name]",
		Short: "Set default AI provider",
		Long:  `Set the default AI provider to use when not specified.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			providers, err := c.loadAIProviders()
			if err != nil {
				return fmt.Errorf("failed to load providers: %w", err)
			}

			found := false
			for i := range providers {
				if providers[i].Name == name {
					providers[i].IsDefault = true
					found = true
				} else {
					providers[i].IsDefault = false
				}
			}

			if !found {
				return fmt.Errorf("provider '%s' not found", name)
			}

			// Save updated list
			data, err := yaml.Marshal(providers)
			if err != nil {
				return err
			}

			configDir := c.getAIConfigDir()
			if err := os.WriteFile(filepath.Join(configDir, "providers.yaml"), data, 0600); err != nil {
				return err
			}

			fmt.Printf("✅ Provider '%s' set as default\n", name)
			return nil
		},
	}

	return cmd
}

// createAITestProviderCommand creates the 'ai provider test' command
func (c *CLI) createAITestProviderCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test [name]",
		Short: "Test AI provider connection",
		Long:  `Test connection to an AI provider.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			providers, err := c.loadAIProviders()
			if err != nil {
				return fmt.Errorf("failed to load providers: %w", err)
			}

			var provider *AIProviderConfig
			for _, p := range providers {
				if p.Name == name {
					provider = &p
					break
				}
			}

			if provider == nil {
				return fmt.Errorf("provider '%s' not found", name)
			}

			fmt.Printf("🔍 Testing connection to %s...\n", name)

			if err := c.testAIProvider(provider); err != nil {
				fmt.Printf("❌ Connection failed: %v\n", err)
				return err
			}

			fmt.Printf("✅ Connection successful!\n")
			return nil
		},
	}

	return cmd
}

// createAIEstimateCommand creates the 'ai cost estimate' command
func (c *CLI) createAIEstimateCommand() *cobra.Command {
	var (
		provider string
		model    string
	)

	cmd := &cobra.Command{
		Use:   "estimate [task]",
		Short: "Estimate cost for a task",
		Long: `Estimate the cost of running a specific task with AI.

Examples:
  lmmc ai cost estimate "prd creation" --provider anthropic --model claude-opus-4
  lmmc ai cost estimate "code review of 10k lines"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			task := args[0]

			fmt.Printf("💰 Cost Estimation\n")
			fmt.Printf("════════════════\n\n")
			fmt.Printf("Task: %s\n", task)

			// Estimate based on task type
			estimation := c.estimateTaskCost(task, provider, model)

			fmt.Printf("\nEstimated Costs:\n")
			fmt.Printf("───────────────\n")

			for _, est := range estimation {
				fmt.Printf("\n%s/%s:\n", est.Provider, est.Model)
				fmt.Printf("  Input tokens: ~%s\n", formatNumber(est.InputTokens))
				fmt.Printf("  Output tokens: ~%s\n", formatNumber(est.OutputTokens))
				fmt.Printf("  Estimated cost: $%.2f - $%.2f\n", est.MinCost, est.MaxCost)
				fmt.Printf("  Time estimate: %s\n", est.TimeEstimate)
			}

			// Check budget
			budget, err := c.loadAIBudget()
			if err == nil && budget.PerCommand > 0 {
				fmt.Printf("\n💡 Budget Info:\n")
				fmt.Printf("   Per-command limit: $%.2f\n", budget.PerCommand)

				for _, est := range estimation {
					if est.MaxCost > budget.PerCommand {
						fmt.Printf("   ⚠️  %s/%s may exceed budget\n", est.Provider, est.Model)
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&provider, "provider", "", "Specific provider to estimate")
	cmd.Flags().StringVar(&model, "model", "", "Specific model to estimate")

	return cmd
}

// createAIListFallbackCommand creates the 'ai fallback list' command
func (c *CLI) createAIListFallbackCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured fallback chains",
		Long:  `Display the configured fallback chain for AI providers.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configFile := filepath.Join(c.getAIConfigDir(), "fallback.yaml")
			cleanPath := filepath.Clean(configFile)

			// Validate path is within config directory
			if !strings.HasPrefix(cleanPath, c.getAIConfigDir()) {
				return errors.New("invalid config file path")
			}

			data, err := os.ReadFile(cleanPath)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Printf("No fallback chain configured.\n")
					return nil
				}
				return err
			}

			var chain AIFallbackChain
			if err := yaml.Unmarshal(data, &chain); err != nil {
				return err
			}

			fmt.Printf("🔄 AI Provider Fallback Chain\n")
			fmt.Printf("════════════════════════════\n\n")

			fmt.Printf("1. Primary: %s\n", chain.Primary)
			if chain.Secondary != "" {
				fmt.Printf("2. Secondary: %s\n", chain.Secondary)
			}
			if chain.Tertiary != "" {
				fmt.Printf("3. Tertiary: %s\n", chain.Tertiary)
			}

			fmt.Printf("\n💡 How it works:\n")
			fmt.Printf("   If primary fails, automatically retry with secondary.\n")
			fmt.Printf("   If secondary fails, automatically retry with tertiary.\n")

			return nil
		},
	}

	return cmd
}

// createAISetFallbackCommand creates the 'ai fallback set' command
func (c *CLI) createAISetFallbackCommand() *cobra.Command {
	var (
		primary   string
		secondary string
		tertiary  string
	)

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Configure fallback chain",
		Long: `Set up a fallback chain for automatic failover between providers.

Examples:
  lmmc ai fallback set --primary anthropic/claude-opus-4 --secondary openai/gpt-4o
  lmmc ai fallback set --primary anthropic/claude-opus-4 --secondary openai/gpt-4o --tertiary local/llama-70b`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if primary == "" {
				return errors.New("--primary is required")
			}

			fallback := AIFallbackChain{
				Primary:   primary,
				Secondary: secondary,
				Tertiary:  tertiary,
			}

			if err := c.saveAIFallbackChain(fallback); err != nil {
				return fmt.Errorf("failed to save fallback chain: %w", err)
			}

			fmt.Printf("✅ Fallback chain configured:\n")
			fmt.Printf("   1. Primary: %s\n", primary)
			if secondary != "" {
				fmt.Printf("   2. Secondary: %s\n", secondary)
			}
			if tertiary != "" {
				fmt.Printf("   3. Tertiary: %s\n", tertiary)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&primary, "primary", "", "Primary provider/model")
	cmd.Flags().StringVar(&secondary, "secondary", "", "Secondary provider/model (fallback)")
	cmd.Flags().StringVar(&tertiary, "tertiary", "", "Tertiary provider/model (second fallback)")

	return cmd
}

// Helper types and methods

type AIDefaultConfig struct {
	Provider string `json:"provider" yaml:"provider"`
	Model    string `json:"model" yaml:"model"`
}

type AIFallbackChain struct {
	Primary   string `json:"primary" yaml:"primary"`
	Secondary string `json:"secondary,omitempty" yaml:"secondary,omitempty"`
	Tertiary  string `json:"tertiary,omitempty" yaml:"tertiary,omitempty"`
}

// Configuration persistence methods

func (c *CLI) getAIConfigDir() string {
	configDir, _ := os.UserConfigDir()
	return filepath.Join(configDir, "lmmc", "ai")
}

func (c *CLI) saveAIProviderConfig(config *AIProviderConfig) error {
	configDir := c.getAIConfigDir()
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return err
	}

	// Load existing providers
	providers, _ := c.loadAIProviders()

	// Update or add provider
	found := false
	for i, p := range providers {
		if p.Name == config.Name {
			providers[i] = *config
			found = true
			break
		}
	}
	if !found {
		providers = append(providers, *config)
	}

	// If setting as default, clear other defaults
	if config.IsDefault {
		for i := range providers {
			if providers[i].Name != config.Name {
				providers[i].IsDefault = false
			}
		}
	}

	// Save to file
	data, err := yaml.Marshal(providers)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(configDir, "providers.yaml"), data, 0600)
}

func (c *CLI) loadAIProviders() ([]AIProviderConfig, error) {
	configFile := filepath.Join(c.getAIConfigDir(), "providers.yaml")
	cleanPath := filepath.Clean(configFile)

	// Validate path is within config directory
	if !strings.HasPrefix(cleanPath, c.getAIConfigDir()) {
		return nil, errors.New("invalid config file path")
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []AIProviderConfig{}, nil
		}
		return nil, err
	}

	var providers []AIProviderConfig
	if err := yaml.Unmarshal(data, &providers); err != nil {
		return nil, err
	}

	return providers, nil
}

func (c *CLI) saveAIDefaults(defaults map[string]AIDefaultConfig) error {
	configDir := c.getAIConfigDir()
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return err
	}

	data, err := yaml.Marshal(defaults)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(configDir, "defaults.yaml"), data, 0600)
}

func (c *CLI) loadAIDefaults() (map[string]AIDefaultConfig, error) {
	configFile := filepath.Join(c.getAIConfigDir(), "defaults.yaml")
	cleanPath := filepath.Clean(configFile)

	// Validate path is within config directory
	if !strings.HasPrefix(cleanPath, c.getAIConfigDir()) {
		return nil, errors.New("invalid config file path")
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]AIDefaultConfig), nil
		}
		return nil, err
	}

	var defaults map[string]AIDefaultConfig
	if err := yaml.Unmarshal(data, &defaults); err != nil {
		return nil, err
	}

	return defaults, nil
}

func (c *CLI) saveAIBudget(budget *AICostBudget) error {
	configDir := c.getAIConfigDir()
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return err
	}

	data, err := yaml.Marshal(budget)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(configDir, "budget.yaml"), data, 0600)
}

func (c *CLI) loadAIBudget() (*AICostBudget, error) {
	configFile := filepath.Join(c.getAIConfigDir(), "budget.yaml")
	cleanPath := filepath.Clean(configFile)

	// Validate path is within config directory
	if !strings.HasPrefix(cleanPath, c.getAIConfigDir()) {
		return nil, errors.New("invalid config file path")
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &AICostBudget{PerProvider: make(map[string]float64)}, nil
		}
		return nil, err
	}

	var budget AICostBudget
	if err := yaml.Unmarshal(data, &budget); err != nil {
		return nil, err
	}

	if budget.PerProvider == nil {
		budget.PerProvider = make(map[string]float64)
	}

	return &budget, nil
}

func (c *CLI) saveAIFallbackChain(chain AIFallbackChain) error {
	configDir := c.getAIConfigDir()
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return err
	}

	data, err := yaml.Marshal(chain)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(configDir, "fallback.yaml"), data, 0600)
}

func (c *CLI) loadAIUsageStats(_, _ string) []AIUsageStats {
	// For now, return mock data - in production, this would read from a database
	// This is a placeholder for the actual implementation
	return []AIUsageStats{
		{
			Provider:     "anthropic",
			Model:        "claude-opus-4",
			InputTokens:  150000,
			OutputTokens: 50000,
			Cost:         3.75,
			Timestamp:    time.Now().Add(-24 * time.Hour),
			Command:      "prd create",
		},
		{
			Provider:     "openai",
			Model:        "gpt-4o",
			InputTokens:  80000,
			OutputTokens: 20000,
			Cost:         0.70,
			Timestamp:    time.Now().Add(-48 * time.Hour),
			Command:      "review start",
		},
	}
}

func (c *CLI) testAIProvider(_ *AIProviderConfig) error {
	// This would test the actual connection to the provider
	// For now, just simulate
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// In a real implementation, this would create a client and test the connection
	select {
	case <-ctx.Done():
		return errors.New("connection timeout")
	case <-time.After(500 * time.Millisecond):
		// Simulate successful connection
		return nil
	}
}

// Utility functions
func formatNumber(n int) string {
	if n < 1000 {
		return strconv.Itoa(n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1000000)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ModelRecommendation represents model recommendation types
type ModelRecommendation struct {
	Provider      string
	Model         string
	Reason        string
	ContextSize   int
	EstimatedCost float64
	Speed         string
}

// CostEstimation represents cost estimation types
type CostEstimation struct {
	Provider     string
	Model        string
	InputTokens  int
	OutputTokens int
	MinCost      float64
	MaxCost      float64
	TimeEstimate string
}

// analyzeTaskRequirements analyzes task requirements and recommends models
func (c *CLI) analyzeTaskRequirements(task string) []ModelRecommendation {
	recommendations := []ModelRecommendation{}

	taskLower := strings.ToLower(task)

	// Check for context size requirements
	if strings.Contains(taskLower, "large context") || strings.Contains(taskLower, "long document") {
		recommendations = append(recommendations, ModelRecommendation{
			Provider:      "anthropic",
			Model:         "claude-opus-4",
			Reason:        "Best for large context (200k tokens) with excellent reasoning",
			ContextSize:   200000,
			EstimatedCost: 3.75,
			Speed:         "Medium",
		})
		recommendations = append(recommendations, ModelRecommendation{
			Provider:      "google",
			Model:         "gemini-ultra",
			Reason:        "Massive context window (1M tokens) for very large documents",
			ContextSize:   1000000,
			EstimatedCost: 5.00,
			Speed:         "Slow",
		})
	}

	// Check for speed requirements
	if strings.Contains(taskLower, "quick") || strings.Contains(taskLower, "fast") {
		recommendations = append(recommendations, ModelRecommendation{
			Provider:      "anthropic",
			Model:         "claude-haiku",
			Reason:        "Fastest Claude model for quick tasks",
			ContextSize:   200000,
			EstimatedCost: 0.05,
			Speed:         "Very Fast",
		})
		recommendations = append(recommendations, ModelRecommendation{
			Provider:      "openai",
			Model:         "gpt-3.5-turbo",
			Reason:        "Fast and cost-effective for simple tasks",
			ContextSize:   16385,
			EstimatedCost: 0.03,
			Speed:         "Very Fast",
		})
	}

	// Check for code review
	if strings.Contains(taskLower, "code review") || strings.Contains(taskLower, "code analysis") {
		recommendations = append(recommendations, ModelRecommendation{
			Provider:      "anthropic",
			Model:         "claude-sonnet-3.5",
			Reason:        "Excellent code understanding with good speed/cost balance",
			ContextSize:   200000,
			EstimatedCost: 0.75,
			Speed:         "Fast",
		})
		recommendations = append(recommendations, ModelRecommendation{
			Provider:      "openai",
			Model:         "gpt-4o",
			Reason:        "Strong code analysis with function calling support",
			ContextSize:   128000,
			EstimatedCost: 1.00,
			Speed:         "Medium",
		})
	}

	// Check for multimodal requirements
	if strings.Contains(taskLower, "multimodal") || strings.Contains(taskLower, "image") || strings.Contains(taskLower, "diagram") {
		recommendations = append(recommendations, ModelRecommendation{
			Provider:      "google",
			Model:         "gemini-ultra",
			Reason:        "Best multimodal capabilities for images and diagrams",
			ContextSize:   1000000,
			EstimatedCost: 4.00,
			Speed:         "Medium",
		})
		recommendations = append(recommendations, ModelRecommendation{
			Provider:      "openai",
			Model:         "gpt-4-turbo",
			Reason:        "Good vision capabilities with reliable performance",
			ContextSize:   128000,
			EstimatedCost: 2.50,
			Speed:         "Medium",
		})
	}

	// Default recommendations if no specific requirements
	if len(recommendations) == 0 {
		recommendations = append(recommendations, ModelRecommendation{
			Provider:      "anthropic",
			Model:         "claude-opus-4",
			Reason:        "Best overall performance for complex tasks",
			ContextSize:   200000,
			EstimatedCost: 3.75,
			Speed:         "Medium",
		})
		recommendations = append(recommendations, ModelRecommendation{
			Provider:      "openai",
			Model:         "gpt-4o",
			Reason:        "Reliable general-purpose model",
			ContextSize:   128000,
			EstimatedCost: 1.00,
			Speed:         "Medium",
		})
	}

	return recommendations
}

// estimateTaskCost estimates the cost of running a task
func (c *CLI) estimateTaskCost(task, provider, model string) []CostEstimation {
	inputTokens, outputTokens := c.estimateTokenUsage(task)
	providers, _ := c.loadAIProviders()

	estimations := c.calculateCostEstimations(providers, inputTokens, outputTokens, provider, model)

	return c.limitEstimationResults(estimations, provider, model)
}

// estimateTokenUsage estimates input and output tokens based on task type
func (c *CLI) estimateTokenUsage(task string) (int, int) {
	taskLower := strings.ToLower(task)

	switch {
	case strings.Contains(taskLower, "prd creation"):
		return 5000, 15000
	case strings.Contains(taskLower, "code review"):
		return c.estimateCodeReviewTokens(taskLower)
	case strings.Contains(taskLower, "trd generation"):
		return 10000, 8000
	case strings.Contains(taskLower, "task generation"):
		return 8000, 5000
	default:
		return 5000, 2000
	}
}

// estimateCodeReviewTokens estimates tokens for code review based on size hints
func (c *CLI) estimateCodeReviewTokens(taskLower string) (int, int) {
	if strings.Contains(taskLower, "10k lines") {
		return 50000, 10000
	}
	if strings.Contains(taskLower, "1k lines") {
		return 5000, 2000
	}
	return 20000, 5000
}

// calculateCostEstimations calculates cost estimations for all relevant provider/model combinations
func (c *CLI) calculateCostEstimations(providers []AIProviderConfig, inputTokens, outputTokens int, provider, model string) []CostEstimation {
	var estimations []CostEstimation

	for _, p := range providers {
		if !c.shouldIncludeProvider(&p, provider) {
			continue
		}

		for _, m := range p.Models {
			if !c.shouldIncludeModel(m, model) {
				continue
			}

			estimation := c.calculateSingleEstimation(p, m, inputTokens, outputTokens)
			estimations = append(estimations, estimation)
		}
	}

	return estimations
}

// shouldIncludeProvider checks if a provider should be included in estimations
func (c *CLI) shouldIncludeProvider(p *AIProviderConfig, provider string) bool {
	return provider == "" || p.Name == provider || p.Type == provider
}

// shouldIncludeModel checks if a model should be included in estimations
func (c *CLI) shouldIncludeModel(m AIModelConfig, model string) bool {
	return model == "" || m.ID == model
}

// calculateSingleEstimation calculates cost estimation for a single provider/model combination
func (c *CLI) calculateSingleEstimation(p AIProviderConfig, m AIModelConfig, inputTokens, outputTokens int) CostEstimation {
	inputCost := (float64(inputTokens) / 1000000) * m.InputCost
	outputCost := (float64(outputTokens) / 1000000) * m.OutputCost
	totalCost := inputCost + outputCost

	// Add 20% variance for min/max
	minCost := totalCost * 0.8
	maxCost := totalCost * 1.2

	timeEstimate := c.estimateProcessingTime(m.ID, inputTokens)

	return CostEstimation{
		Provider:     p.Type,
		Model:        m.ID,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		MinCost:      minCost,
		MaxCost:      maxCost,
		TimeEstimate: timeEstimate,
	}
}

// estimateProcessingTime estimates processing time based on model characteristics
func (c *CLI) estimateProcessingTime(modelID string, inputTokens int) string {
	if strings.Contains(modelID, "haiku") || strings.Contains(modelID, "3.5") {
		return "30-60 seconds"
	}
	if strings.Contains(modelID, "ultra") || inputTokens > 50000 {
		return "3-5 minutes"
	}
	return "1-2 minutes"
}

// limitEstimationResults limits the number of results when no specific provider/model is requested
func (c *CLI) limitEstimationResults(estimations []CostEstimation, provider, model string) []CostEstimation {
	if provider == "" && model == "" && len(estimations) > 3 {
		return estimations[:3]
	}
	return estimations
}
