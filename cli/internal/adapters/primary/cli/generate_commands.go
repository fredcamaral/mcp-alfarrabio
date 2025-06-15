package cli

import (
	cryptorand "crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"lerian-mcp-memory-cli/internal/domain/constants"
	"lerian-mcp-memory-cli/internal/domain/services"
)

// createGenerateCommand creates the generate command group
func (c *CLI) createGenerateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate sample data for testing and demos",
		Long: `Generate sample data including tasks, PRDs, TRDs, and complete projects.
		
This is useful for:
- Testing the CLI with realistic data
- Creating demo environments
- Learning how to use the system
- Performance testing with large datasets`,
	}

	// Add subcommands
	cmd.AddCommand(
		c.createGenerateSampleTasksCommand(),
		c.createGenerateSamplePRDCommand(),
		c.createGenerateSampleProjectCommand(),
	)

	return cmd
}

// createGenerateSampleTasksCommand generates sample tasks
func (c *CLI) createGenerateSampleTasksCommand() *cobra.Command {
	var (
		count        int
		repository   string
		priority     string
		tags         []string
		withSubtasks bool
	)

	cmd := &cobra.Command{
		Use:   "sample-tasks",
		Short: "Generate sample tasks for testing",
		Long:  `Generate a specified number of sample tasks with realistic content.`,
		Example: `  # Generate 20 sample tasks
  lmmc generate sample-tasks --count 20
  
  # Generate high priority tasks with tags
  lmmc generate sample-tasks --count 10 --priority high --tags backend,api
  
  # Generate tasks with subtasks
  lmmc generate sample-tasks --count 5 --with-subtasks`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runGenerateSampleTasks(cmd, count, repository, priority, tags, withSubtasks)
		},
	}

	cmd.Flags().IntVarP(&count, "count", "c", 10, "Number of tasks to generate")
	cmd.Flags().StringVarP(&repository, "repository", "r", "", "Repository name (auto-detected if not specified)")
	cmd.Flags().StringVarP(&priority, "priority", "p", "", "Task priority (low, medium, high, or random)")
	cmd.Flags().StringSliceVarP(&tags, "tags", "t", nil, "Tags to add to all tasks")
	cmd.Flags().BoolVar(&withSubtasks, "with-subtasks", false, "Generate subtasks for each main task")

	return cmd
}

// createGenerateSamplePRDCommand generates a sample PRD
func (c *CLI) createGenerateSamplePRDCommand() *cobra.Command {
	var (
		projectType string
		output      string
		features    int
	)

	cmd := &cobra.Command{
		Use:   "sample-prd",
		Short: "Generate a sample PRD document",
		Long:  `Generate a sample Product Requirements Document for a specific project type.`,
		Example: `  # Generate e-commerce PRD
  lmmc generate sample-prd --type e-commerce
  
  # Generate API service PRD with custom output
  lmmc generate sample-prd --type api --output custom-prd.md
  
  # Generate PRD with specific number of features
  lmmc generate sample-prd --type web-app --features 10`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runGenerateSamplePRD(cmd, projectType, output, features)
		},
	}

	cmd.Flags().StringVarP(&projectType, "type", "t", constants.ProjectTypeWebApp, "Project type (e-commerce, "+constants.ProjectTypeAPI+", "+constants.ProjectTypeWebApp+", mobile, cli, microservice)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (default: "+constants.DefaultPreDevelopmentDir+"/sample-prd-<type>.md)")
	cmd.Flags().IntVar(&features, "features", 5, "Number of features to include")

	return cmd
}

// createGenerateSampleProjectCommand generates a complete sample project
func (c *CLI) createGenerateSampleProjectCommand() *cobra.Command {
	var (
		template   string
		name       string
		outputDir  string
		withTasks  bool
		withMemory bool
	)

	cmd := &cobra.Command{
		Use:   "sample-project",
		Short: "Generate a complete sample project",
		Long:  `Generate a complete project structure with PRD, TRD, tasks, and optionally memory data.`,
		Example: `  # Generate microservice project
  lmmc generate sample-project --template microservice
  
  # Generate project with custom name
  lmmc generate sample-project --template api --name payment-service
  
  # Generate project with tasks and memory
  lmmc generate sample-project --template web-app --with-tasks --with-memory`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runGenerateSampleProject(cmd, template, name, outputDir, withTasks, withMemory)
		},
	}

	cmd.Flags().StringVarP(&template, "template", "t", constants.ProjectTypeAPI, "Project template ("+constants.ProjectTypeAPI+", microservice, "+constants.ProjectTypeWebApp+", cli, library)")
	cmd.Flags().StringVarP(&name, "name", "n", "", "Project name (auto-generated if not specified)")
	cmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory (default: ./<project-name>)")
	cmd.Flags().BoolVar(&withTasks, "with-tasks", true, "Generate tasks from PRD/TRD")
	cmd.Flags().BoolVar(&withMemory, "with-memory", false, "Store generated content in memory")

	return cmd
}

// Implementation functions

func (c *CLI) runGenerateSampleTasks(cmd *cobra.Command, count int, repository, priority string, tags []string, withSubtasks bool) error {
	// Sample task templates
	taskTemplates := []struct {
		content  string
		tags     []string
		priority string
	}{
		{"Implement user authentication with JWT tokens", []string{"backend", "security"}, "high"},
		{"Add input validation for API endpoints", []string{"backend", "security"}, "high"},
		{"Create unit tests for payment processing", []string{"testing", "backend"}, "medium"},
		{"Optimize database queries for better performance", []string{"backend", "performance"}, "medium"},
		{"Update API documentation with new endpoints", []string{"docs", "api"}, "low"},
		{"Fix CORS issues in production environment", []string{"bug", "backend"}, "high"},
		{"Implement rate limiting for API endpoints", []string{"backend", "security"}, "medium"},
		{"Add monitoring and alerting for critical services", []string{"devops", "monitoring"}, "high"},
		{"Refactor user service to improve maintainability", []string{"refactor", "backend"}, "low"},
		{"Create integration tests for external APIs", []string{"testing", "integration"}, "medium"},
		{"Implement caching layer for frequently accessed data", []string{"backend", "performance"}, "medium"},
		{"Add error handling for edge cases", []string{"backend", "quality"}, "medium"},
		{"Update dependencies to latest stable versions", []string{"maintenance", "security"}, "low"},
		{"Create database migration scripts", []string{"database", "backend"}, "high"},
		{"Implement webhook system for event notifications", []string{"backend", "feature"}, "medium"},
		{"Add support for multiple languages (i18n)", []string{"frontend", "feature"}, "low"},
		{"Improve error messages for better UX", []string{"frontend", "ux"}, "low"},
		{"Set up CI/CD pipeline for automated deployments", []string{"devops", "automation"}, "high"},
		{"Create admin dashboard for system monitoring", []string{"frontend", "feature"}, "medium"},
		{"Implement data export functionality", []string{"backend", "feature"}, "low"},
	}

	// Get repository if not specified
	if repository == "" {
		if repoInfo, err := c.repositoryDetector.DetectCurrent(c.getContext()); err == nil && repoInfo != nil {
			repository = repoInfo.Name
		} else {
			repository = "sample-project"
		}
	}

	// Generate tasks
	for i := 0; i < count; i++ {
		// Pick a random template
		n, _ := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(len(taskTemplates))))
		template := taskTemplates[n.Int64()]

		// Determine priority
		taskPriority := priority
		if taskPriority == "" || taskPriority == "random" {
			priorities := []string{"low", "medium", "high"}
			n, _ := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(len(priorities))))
			taskPriority = priorities[n.Int64()]
		}

		// Combine tags
		taskTags := append(template.tags, tags...)

		// Create task with some variation
		taskContent := fmt.Sprintf("%s (#%d)", template.content, i+1)

		// Build options
		var options []services.TaskOption
		p, _ := parsePriority(taskPriority)
		options = append(options, services.WithPriority(p))
		options = append(options, services.WithTags(taskTags...))

		// Add estimated time
		n1, _ := cryptorand.Int(cryptorand.Reader, big.NewInt(240))
		estimatedTime := 30 + int(n1.Int64()) // 30 min to 4 hours
		options = append(options, services.WithEstimatedTime(estimatedTime))

		// Create the task
		task, err := c.taskService.CreateTask(c.getContext(), taskContent, options...)
		if err != nil {
			return fmt.Errorf("failed to create task %d: %w", i+1, err)
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created task %d/%d: %s\n", i+1, count, task.ID[:8])

		// Generate subtasks if requested
		n3, _ := cryptorand.Int(cryptorand.Reader, big.NewInt(10))
		if withSubtasks && n3.Int64() < 6 { // 60% chance of having subtasks
			n4, _ := cryptorand.Int(cryptorand.Reader, big.NewInt(4))
			subtaskCount := 2 + int(n4.Int64()) // 2-5 subtasks
			for j := 0; j < subtaskCount; j++ {
				subtaskContent := fmt.Sprintf("Subtask %d for %s", j+1, taskContent)
				subtask, err := c.taskService.CreateTask(c.getContext(), subtaskContent,
					services.WithTags("subtask", "parent-"+task.ID[:8]),
					services.WithPriority(p))
				if err != nil {
					c.logger.Warn("Failed to create subtask", "error", err)
				} else {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  └─ Created subtask: %s\n", subtask.ID[:8])
				}
			}
		}
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n✅ Generated %d sample tasks in repository: %s\n", count, repository)
	return nil
}

func (c *CLI) runGenerateSamplePRD(cmd *cobra.Command, projectType, output string, features int) error {
	// PRD templates by project type
	prdTemplates := map[string]struct {
		title       string
		description string
		features    []string
		tech        []string
	}{
		"e-commerce": {
			title:       "E-Commerce Platform PRD",
			description: "A modern e-commerce platform with advanced features for online retail",
			features: []string{
				"User registration and authentication",
				"Product catalog with search and filters",
				"Shopping cart and checkout process",
				"Payment gateway integration",
				"Order tracking and management",
				"Inventory management system",
				"Customer reviews and ratings",
				"Wishlist functionality",
				"Email notifications",
				"Admin dashboard",
				"Analytics and reporting",
				"Mobile responsive design",
			},
			tech: []string{"React", "Node.js", "PostgreSQL", "Redis", "Stripe API"},
		},
		"api": {
			title:       "REST API Service PRD",
			description: "A scalable REST API service for modern applications",
			features: []string{
				"RESTful API endpoints",
				"JWT authentication",
				"Rate limiting",
				"API versioning",
				"Request validation",
				"Error handling",
				"API documentation",
				"Webhook support",
				"Caching layer",
				"Database integration",
				"Monitoring and logging",
				"Health check endpoints",
			},
			tech: []string{"Go", "PostgreSQL", "Redis", "Docker", "Swagger"},
		},
		"web-app": {
			title:       "Web Application PRD",
			description: "A modern web application with rich user interface",
			features: []string{
				"User authentication",
				"Dashboard interface",
				"Real-time notifications",
				"File upload/download",
				"Search functionality",
				"User profiles",
				"Settings management",
				"Data visualization",
				"Export capabilities",
				"Responsive design",
				"Dark mode support",
				"Accessibility features",
			},
			tech: []string{"React", "TypeScript", "Node.js", "MongoDB", "WebSocket"},
		},
		"mobile": {
			title:       "Mobile Application PRD",
			description: "A cross-platform mobile application",
			features: []string{
				"User onboarding",
				"Push notifications",
				"Offline mode",
				"Biometric authentication",
				"Location services",
				"Camera integration",
				"Social sharing",
				"In-app purchases",
				"Analytics tracking",
				"App settings",
				"Multi-language support",
				"Gesture controls",
			},
			tech: []string{"React Native", "Firebase", "Redux", "REST API"},
		},
		"cli": {
			title:       "CLI Tool PRD",
			description: "A command-line interface tool for developers",
			features: []string{
				"Command structure",
				"Configuration management",
				"Plugin system",
				"Auto-completion",
				"Help documentation",
				"Version management",
				"Update mechanism",
				"Logging system",
				"Error handling",
				"Progress indicators",
				"Interactive mode",
				"Batch operations",
			},
			tech: []string{"Go", "Cobra", "Viper", "SQLite"},
		},
		"microservice": {
			title:       "Microservice Architecture PRD",
			description: "A microservice-based system architecture",
			features: []string{
				"Service discovery",
				"API gateway",
				"Message queue",
				"Service mesh",
				"Circuit breakers",
				"Distributed tracing",
				"Centralized logging",
				"Container orchestration",
				"Health monitoring",
				"Auto-scaling",
				"Service authentication",
				"Configuration service",
			},
			tech: []string{"Kubernetes", "Docker", "gRPC", "Kafka", "Istio"},
		},
	}

	// Get template
	template, exists := prdTemplates[projectType]
	if !exists {
		return fmt.Errorf("unknown project type: %s", projectType)
	}

	// Determine output path
	if output == "" {
		if err := os.MkdirAll(constants.DefaultPreDevelopmentDir, 0750); err != nil {
			return fmt.Errorf("failed to create docs directory: %w", err)
		}
		output = fmt.Sprintf("%s/sample-prd-%s.md", constants.DefaultPreDevelopmentDir, projectType)
	}

	// Generate PRD content
	var content strings.Builder
	content.WriteString(fmt.Sprintf("# %s\n\n", template.title))
	content.WriteString(fmt.Sprintf("**Generated**: %s\n\n", time.Now().Format("2006-01-02")))
	content.WriteString("## Executive Summary\n\n")
	content.WriteString(template.description + ". This document outlines the requirements and specifications for building a comprehensive solution.\n\n")

	content.WriteString("## Project Overview\n\n")
	content.WriteString("### Vision\n")
	content.WriteString(fmt.Sprintf("To create a %s that sets new standards in user experience and functionality.\n\n", strings.ToLower(template.title)))

	content.WriteString("### Goals\n")
	content.WriteString("- Deliver a robust and scalable solution\n")
	content.WriteString("- Ensure excellent user experience\n")
	content.WriteString("- Maintain high performance standards\n")
	content.WriteString("- Enable easy maintenance and updates\n\n")

	content.WriteString("## Features and Requirements\n\n")

	// Add selected number of features
	featureCount := features
	if featureCount > len(template.features) {
		featureCount = len(template.features)
	}

	for i := 0; i < featureCount; i++ {
		content.WriteString(fmt.Sprintf("### %d. %s\n", i+1, template.features[i]))
		content.WriteString("**Priority**: High\n\n")
		content.WriteString("**Description**: ")

		// Add some generic description based on feature
		switch {
		case strings.Contains(strings.ToLower(template.features[i]), "auth"):
			content.WriteString("Secure user authentication system with support for multiple authentication methods.\n\n")
		case strings.Contains(strings.ToLower(template.features[i]), "api"):
			content.WriteString("Well-designed API with comprehensive documentation and versioning support.\n\n")
		case strings.Contains(strings.ToLower(template.features[i]), "dashboard"):
			content.WriteString("Intuitive dashboard providing key metrics and insights at a glance.\n\n")
		default:
			content.WriteString(fmt.Sprintf("Implementation of %s with focus on usability and performance.\n\n", template.features[i]))
		}

		content.WriteString("**Acceptance Criteria**:\n")
		content.WriteString(fmt.Sprintf("- %s is fully functional\n", template.features[i]))
		content.WriteString("- All edge cases are handled\n")
		content.WriteString("- Feature is tested and documented\n\n")
	}

	content.WriteString("## Technical Requirements\n\n")
	content.WriteString(buildTechListMarkdown(template.tech))

	content.WriteString("### Non-Functional Requirements\n")
	content.WriteString("- **Performance**: Response time < 200ms for 95% of requests\n")
	content.WriteString("- **Scalability**: Support for 10,000+ concurrent users\n")
	content.WriteString("- **Security**: Industry-standard security practices\n")
	content.WriteString("- **Availability**: 99.9% uptime SLA\n\n")

	content.WriteString("## Timeline\n\n")
	content.WriteString("- **Phase 1** (Weeks 1-4): Core functionality\n")
	content.WriteString("- **Phase 2** (Weeks 5-8): Advanced features\n")
	content.WriteString("- **Phase 3** (Weeks 9-10): Testing and optimization\n")
	content.WriteString("- **Phase 4** (Weeks 11-12): Deployment and launch\n\n")

	content.WriteString("## Success Metrics\n\n")
	content.WriteString("- User adoption rate > 80%\n")
	content.WriteString("- System performance meets all targets\n")
	content.WriteString("- Zero critical bugs in production\n")
	content.WriteString("- Positive user feedback score > 4.5/5\n")

	// Write to file
	if err := os.WriteFile(output, []byte(content.String()), 0600); err != nil {
		return fmt.Errorf("failed to write PRD: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✅ Generated sample PRD: %s\n", output)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "   Type: %s\n", projectType)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "   Features: %d\n", featureCount)

	return nil
}

func (c *CLI) runGenerateSampleProject(cmd *cobra.Command, template, name, outputDir string, withTasks, withMemory bool) error {
	projectConfig := c.setupProjectConfiguration(template, name, outputDir)

	if err := c.createProjectStructure(projectConfig); err != nil {
		return err
	}

	c.printProjectHeader(cmd, projectConfig)

	filePaths, err := c.generateProjectFiles(cmd, projectConfig)
	if err != nil {
		return err
	}

	if withMemory {
		c.storeProjectInMemory(cmd, projectConfig, filePaths)
	}

	if withTasks {
		c.generateProjectTasks(cmd, projectConfig)
	}

	c.printProjectSummary(cmd, projectConfig)
	return nil
}

// Helper functions for runGenerateSampleProject

type ProjectConfig struct {
	Template  string
	Name      string
	OutputDir string
}

type ProjectFilePaths struct {
	PRDPath       string
	TRDPath       string
	ReadmePath    string
	GitignorePath string
}

func (c *CLI) setupProjectConfiguration(template, name, outputDir string) *ProjectConfig {
	if name == "" {
		name = c.generateRandomProjectName()
	}

	if outputDir == "" {
		outputDir = name
	}

	return &ProjectConfig{
		Template:  template,
		Name:      name,
		OutputDir: outputDir,
	}
}

func (c *CLI) generateRandomProjectName() string {
	adjectives := []string{"awesome", "stellar", "quantum", "nexus", "phoenix", "titan"}
	nouns := []string{"api", "service", "platform", "system", "hub", "engine"}

	n1, _ := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(len(adjectives))))
	n2, _ := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(len(nouns))))

	return fmt.Sprintf("%s-%s", adjectives[n1.Int64()], nouns[n2.Int64()])
}

func (c *CLI) createProjectStructure(config *ProjectConfig) error {
	dirs := []string{
		filepath.Join(config.OutputDir, "docs", "pre-development"),
		filepath.Join(config.OutputDir, "docs", "tasks"),
		filepath.Join(config.OutputDir, "src"),
		filepath.Join(config.OutputDir, "tests"),
		filepath.Join(config.OutputDir, "config"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

func (c *CLI) printProjectHeader(cmd *cobra.Command, config *ProjectConfig) {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "🚀 Generating sample project: %s\n", config.Name)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "   Template: %s\n", config.Template)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "   Directory: %s\n\n", config.OutputDir)
}

func (c *CLI) generateProjectFiles(cmd *cobra.Command, config *ProjectConfig) (*ProjectFilePaths, error) {
	paths := &ProjectFilePaths{
		PRDPath:       filepath.Join(config.OutputDir, "docs", "pre-development", "prd.md"),
		TRDPath:       filepath.Join(config.OutputDir, "docs", "pre-development", "trd.md"),
		ReadmePath:    filepath.Join(config.OutputDir, "README.md"),
		GitignorePath: filepath.Join(config.OutputDir, ".gitignore"),
	}

	// Generate PRD
	if err := c.runGenerateSamplePRD(cmd, config.Template, paths.PRDPath, 8); err != nil {
		return nil, fmt.Errorf("failed to generate PRD: %w", err)
	}

	// Generate TRD
	if err := generateSampleTRD(config.Template, config.Name, paths.TRDPath); err != nil {
		return nil, fmt.Errorf("failed to generate TRD: %w", err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✅ Generated sample TRD: %s\n", paths.TRDPath)

	// Generate README
	if err := generateProjectReadme(config.Template, config.Name, paths.ReadmePath); err != nil {
		return nil, fmt.Errorf("failed to generate README: %w", err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✅ Generated README: %s\n", paths.ReadmePath)

	// Generate .gitignore
	if err := generateGitignore(config.Template, paths.GitignorePath); err != nil {
		return nil, fmt.Errorf("failed to generate .gitignore: %w", err)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✅ Generated .gitignore: %s\n", paths.GitignorePath)

	return paths, nil
}

func (c *CLI) storeProjectInMemory(cmd *cobra.Command, config *ProjectConfig, paths *ProjectFilePaths) {
	if c.getMCPClient() == nil {
		return
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n📝 Storing project documents in memory...\n")
	sessionID := fmt.Sprintf("sample-project-%d", time.Now().Unix())

	c.storeFileInMemory(cmd, paths.PRDPath, "prd", config.Name, sessionID, config.Template)
	c.storeFileInMemory(cmd, paths.TRDPath, "trd", config.Name, sessionID, config.Template)
}

func (c *CLI) storeFileInMemory(cmd *cobra.Command, filePath, fileType, repository, sessionID, template string) {
	content, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return
	}

	_, err = c.getMCPClient().CallMCPTool(c.getContext(), "memory_create", map[string]interface{}{
		"operation": "store_chunk",
		"scope":     "single",
		"options": map[string]interface{}{
			"repository": repository,
			"session_id": sessionID,
			"content":    string(content),
			"metadata": map[string]interface{}{
				"type":     fileType,
				"filename": filepath.Base(filePath),
				"tags":     []string{"sample", "generated", template},
			},
		},
	})

	if err == nil {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "   ✓ Stored %s in memory\n", strings.ToUpper(fileType))
	}
}

func (c *CLI) generateProjectTasks(cmd *cobra.Command, config *ProjectConfig) {
	originalDir, _ := os.Getwd()
	if err := os.Chdir(config.OutputDir); err != nil {
		c.logger.Warn("Failed to change directory", "error", err)
		return
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			c.logger.Warn("Failed to restore directory", "error", err)
		}
	}()

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n🔧 Generating tasks from PRD/TRD...\n")

	n, _ := cryptorand.Int(cryptorand.Reader, big.NewInt(10))
	taskCount := 15 + int(n.Int64()) // 15-25 tasks

	if err := c.runGenerateSampleTasks(cmd, taskCount, config.Name, "", []string{config.Template, "generated"}, true); err != nil {
		c.logger.Warn("Failed to generate tasks", "error", err)
	}
}

func (c *CLI) printProjectSummary(cmd *cobra.Command, config *ProjectConfig) {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\n✨ Sample project generated successfully!\n")
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "   Name: %s\n", config.Name)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "   Location: %s\n", config.OutputDir)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nNext steps:\n")
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "   cd %s\n", config.OutputDir)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "   lmmc list\n")
}

// Helper functions

func generateSampleTRD(projectType, projectName, path string) error {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("# Technical Requirements Document - %s\n\n", projectName))
	content.WriteString(fmt.Sprintf("**Generated**: %s\n\n", time.Now().Format("2006-01-02")))

	content.WriteString("## Architecture Overview\n\n")

	switch projectType {
	case constants.ProjectTypeAPI, "microservice":
		content.WriteString("### System Architecture\n")
		content.WriteString("- **Pattern**: RESTful API / Microservices\n")
		content.WriteString("- **Communication**: HTTP/JSON, gRPC for internal services\n")
		content.WriteString("- **Database**: PostgreSQL with Redis caching\n")
		content.WriteString("- **Deployment**: Docker containers on Kubernetes\n\n")

	case constants.ProjectTypeWebApp, "e-commerce":
		content.WriteString("### System Architecture\n")
		content.WriteString("- **Frontend**: React with TypeScript\n")
		content.WriteString("- **Backend**: Node.js with Express\n")
		content.WriteString("- **Database**: PostgreSQL with Redis caching\n")
		content.WriteString("- **Deployment**: Docker containers with nginx\n\n")

	default:
		content.WriteString("### System Architecture\n")
		content.WriteString("- **Pattern**: Modular architecture\n")
		content.WriteString("- **Components**: Core, Services, Interface\n")
		content.WriteString("- **Storage**: Configurable backend\n")
		content.WriteString("- **Deployment**: Flexible deployment options\n\n")
	}

	content.WriteString("## Technical Specifications\n\n")
	content.WriteString("### API Design\n")
	content.WriteString("- RESTful endpoints following OpenAPI 3.0\n")
	content.WriteString("- JSON request/response format\n")
	content.WriteString("- Versioned APIs (v1, v2)\n")
	content.WriteString("- Standard HTTP status codes\n\n")

	content.WriteString("### Security\n")
	content.WriteString("- JWT-based authentication\n")
	content.WriteString("- HTTPS enforcement\n")
	content.WriteString("- Rate limiting per endpoint\n")
	content.WriteString("- Input validation and sanitization\n\n")

	content.WriteString("### Database Schema\n")
	content.WriteString("```sql\n")
	content.WriteString("-- Core tables\n")
	content.WriteString("CREATE TABLE users (\n")
	content.WriteString("    id UUID PRIMARY KEY,\n")
	content.WriteString("    email VARCHAR(255) UNIQUE NOT NULL,\n")
	content.WriteString("    created_at TIMESTAMP DEFAULT NOW()\n")
	content.WriteString(");\n")
	content.WriteString("```\n\n")

	content.WriteString("### Development Guidelines\n")
	content.WriteString("- Follow clean code principles\n")
	content.WriteString("- Write unit tests (80% coverage)\n")
	content.WriteString("- Document all public APIs\n")
	content.WriteString("- Use semantic versioning\n\n")

	content.WriteString("## Implementation Plan\n")
	content.WriteString("1. Set up development environment\n")
	content.WriteString("2. Implement core functionality\n")
	content.WriteString("3. Add authentication layer\n")
	content.WriteString("4. Build API endpoints\n")
	content.WriteString("5. Implement data layer\n")
	content.WriteString("6. Add monitoring and logging\n")
	content.WriteString("7. Write tests\n")
	content.WriteString("8. Deploy to staging\n")

	return os.WriteFile(path, []byte(content.String()), 0600)
}

func generateProjectReadme(projectType, projectName, path string) error {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("# %s\n\n", projectName))
	content.WriteString(fmt.Sprintf("A sample %s project generated by lmmc.\n\n", projectType))

	content.WriteString("## Overview\n\n")
	content.WriteString("This is a sample project demonstrating the structure and documentation ")
	content.WriteString("generated by the Lerian MCP Memory CLI tool.\n\n")

	content.WriteString("## Project Structure\n\n")
	content.WriteString("```\n")
	content.WriteString(projectName + "/\n")
	content.WriteString("├── docs/\n")
	content.WriteString("│   └── pre-development/\n")
	content.WriteString("│       ├── prd.md          # Product Requirements\n")
	content.WriteString("│       └── trd.md          # Technical Requirements\n")
	content.WriteString("├── src/                    # Source code\n")
	content.WriteString("├── tests/                  # Test files\n")
	content.WriteString("├── config/                 # Configuration\n")
	content.WriteString("└── README.md              # This file\n")
	content.WriteString("```\n\n")

	content.WriteString("## Getting Started\n\n")
	content.WriteString("1. Review the PRD and TRD documents\n")
	content.WriteString("2. Check the generated tasks: `lmmc list`\n")
	content.WriteString("3. Start working on tasks: `lmmc start <task-id>`\n\n")

	content.WriteString("## Generated with lmmc\n\n")
	content.WriteString("This project was generated using:\n")
	content.WriteString("```bash\n")
	content.WriteString(fmt.Sprintf("lmmc generate sample-project --template %s --name %s\n", projectType, projectName))
	content.WriteString("```\n")

	return os.WriteFile(path, []byte(content.String()), 0600)
}

func generateGitignore(projectType, path string) error {
	var content strings.Builder

	// Common ignores
	content.WriteString("# Common\n")
	content.WriteString(".DS_Store\n")
	content.WriteString("*.log\n")
	content.WriteString("*.tmp\n")
	content.WriteString(".env\n")
	content.WriteString(".env.local\n\n")

	// Language-specific
	switch projectType {
	case constants.ProjectTypeAPI, "microservice", "cli":
		content.WriteString("# Go\n")
		content.WriteString("*.exe\n")
		content.WriteString("*.dll\n")
		content.WriteString("*.so\n")
		content.WriteString("*.dylib\n")
		content.WriteString("bin/\n")
		content.WriteString("vendor/\n\n")

	case constants.ProjectTypeWebApp, "e-commerce":
		content.WriteString("# Node\n")
		content.WriteString("node_modules/\n")
		content.WriteString("dist/\n")
		content.WriteString("build/\n")
		content.WriteString(".cache/\n")
		content.WriteString("*.lock\n\n")
	}

	// IDE
	content.WriteString("# IDE\n")
	content.WriteString(".idea/\n")
	content.WriteString(".vscode/\n")
	content.WriteString("*.swp\n")
	content.WriteString("*.swo\n")

	return os.WriteFile(path, []byte(content.String()), 0600)
}
