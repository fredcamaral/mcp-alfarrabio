package repl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"lerian-mcp-memory/internal/ai"
	"lerian-mcp-memory/internal/documents"
)

// handleImportCommand handles document import
func (r *REPL) handleImportCommand(ctx context.Context, docType, filename string) (string, error) {
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found: %s", filename)
	}

	switch docType {
	case "prd":
		return r.importPRD(ctx, filename)
	case "trd":
		return r.importTRD(ctx, filename)
	default:
		return "", fmt.Errorf("unsupported document type for import: %s", docType)
	}
}

// importPRD imports a PRD from file
func (r *REPL) importPRD(ctx context.Context, filename string) (string, error) {
	r.printInfo(fmt.Sprintf("Importing PRD from: %s", filename))

	// Process the PRD file
	prd, err := r.processor.ProcessPRDFile(filename, r.session.Repository)
	if err != nil {
		return "", fmt.Errorf("failed to process PRD: %w", err)
	}

	// Validate PRD
	if err := r.processor.ValidatePRD(prd); err != nil {
		r.printInfo(fmt.Sprintf("Warning: PRD validation issues: %v", err))
	}

	// Store in context
	r.session.mu.Lock()
	r.session.Context["current_prd"] = prd
	r.session.Context["prd_filename"] = filename
	r.session.UpdatedAt = time.Now()
	r.session.mu.Unlock()

	// Send notification
	r.sendNotification(Notification{
		Type:    "document_imported",
		Message: fmt.Sprintf("PRD imported: %s", prd.Title),
		Data: map[string]interface{}{
			"document_type": "prd",
			"document_id":   prd.ID,
			"complexity":    prd.ComplexityScore,
		},
	})

	return fmt.Sprintf("PRD imported successfully: %s (Complexity: %d/100)", prd.Title, prd.ComplexityScore), nil
}

// importTRD imports a TRD from file
func (r *REPL) importTRD(ctx context.Context, filename string) (string, error) {
	// Check if we have a current PRD
	r.session.mu.RLock()
	currentPRD, hasPRD := r.session.Context["current_prd"].(*documents.PRDEntity)
	r.session.mu.RUnlock()

	if !hasPRD {
		return "", fmt.Errorf("no PRD loaded - import or create a PRD first")
	}

	r.printInfo(fmt.Sprintf("Importing TRD from: %s", filename))

	// Read file content
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read TRD file: %w", err)
	}

	// Create TRD entity
	trd := &documents.TRDEntity{
		PRDID:      currentPRD.ID,
		Content:    string(content),
		Repository: r.session.Repository,
		Status:     documents.StatusDraft,
		Metadata:   make(map[string]string),
	}

	// Parse sections
	trd.Sections = documents.ParseSections(string(content))
	if len(trd.Sections) > 0 {
		trd.Title = trd.Sections[0].Title
	}

	// Extract technical stack
	trd.TechnicalStack = extractTechnicalStack(string(content))
	trd.Architecture = detectArchitecture(string(content))

	// Validate
	if err := trd.Validate(); err != nil {
		return "", fmt.Errorf("TRD validation failed: %w", err)
	}

	// Store in context
	r.session.mu.Lock()
	r.session.Context["current_trd"] = trd
	r.session.Context["trd_filename"] = filename
	r.session.UpdatedAt = time.Now()
	r.session.mu.Unlock()

	return fmt.Sprintf("TRD imported successfully: %s", trd.Title), nil
}

// handleGenerateCommand handles document generation
func (r *REPL) handleGenerateCommand(ctx context.Context, docType string, args []string) (string, error) {
	switch docType {
	case "trd":
		return r.generateTRD(ctx)
	case "tasks":
		return r.generateTasks(ctx)
	case "subtasks":
		if len(args) == 0 {
			return "", fmt.Errorf("specify main task ID")
		}
		return r.generateSubTasks(ctx, args[0])
	default:
		return "", fmt.Errorf("unsupported generation type: %s", docType)
	}
}

// generateTRD generates a TRD from current PRD
func (r *REPL) generateTRD(ctx context.Context) (string, error) {
	// Check if we have a current PRD
	r.session.mu.RLock()
	currentPRD, hasPRD := r.session.Context["current_prd"].(*documents.PRDEntity)
	r.session.mu.RUnlock()

	if !hasPRD {
		return "", fmt.Errorf("no PRD loaded - import or create a PRD first")
	}

	r.printInfo("Generating TRD from current PRD...")

	// Create generation request
	req := &ai.DocumentGenerationRequest{
		Type:       ai.DocumentTypeTRD,
		Repository: r.session.Repository,
		SessionID:  r.session.ID,
		SourcePRD:  currentPRD,
		Context: map[string]string{
			"prd_title":      currentPRD.Title,
			"prd_complexity": fmt.Sprintf("%d", currentPRD.ComplexityScore),
		},
	}

	// Generate TRD
	resp, err := r.documentGen.GenerateDocument(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to generate TRD: %w", err)
	}

	// Store in context
	r.session.mu.Lock()
	r.session.Context["current_trd"] = resp.Document
	r.session.UpdatedAt = time.Now()
	r.session.mu.Unlock()

	// Show suggestions
	if len(resp.Suggestions) > 0 {
		r.printInfo("Suggestions:")
		for _, suggestion := range resp.Suggestions {
			r.printInfo(fmt.Sprintf("  - %s", suggestion))
		}
	}

	return fmt.Sprintf("TRD generated successfully (Model: %s, Tokens: %d)", resp.ModelUsed, resp.TokensUsed.Total), nil
}

// generateTasks generates main tasks from PRD and TRD
func (r *REPL) generateTasks(ctx context.Context) (string, error) {
	// Check if we have PRD and TRD
	r.session.mu.RLock()
	currentPRD, hasPRD := r.session.Context["current_prd"].(*documents.PRDEntity)
	currentTRD, hasTRD := r.session.Context["current_trd"].(*documents.TRDEntity)
	r.session.mu.RUnlock()

	if !hasPRD || !hasTRD {
		return "", fmt.Errorf("both PRD and TRD are required for task generation")
	}

	r.printInfo("Generating main tasks from PRD and TRD...")

	// Generate tasks using task generator
	mainTasks, err := r.taskGen.GenerateMainTasks(currentPRD, currentTRD)
	if err != nil {
		return "", fmt.Errorf("failed to generate tasks: %w", err)
	}

	// Store in context
	r.session.mu.Lock()
	r.session.Context["main_tasks"] = mainTasks
	r.session.UpdatedAt = time.Now()
	r.session.mu.Unlock()

	// Display task summary
	r.printInfo(fmt.Sprintf("Generated %d main tasks:", len(mainTasks)))
	for _, task := range mainTasks {
		r.printInfo(fmt.Sprintf("  %s: %s (%s)", task.TaskID, task.Name, task.DurationEstimate))
	}

	// Generate dependency graph
	graph := documents.GenerateTaskDependencyGraph(mainTasks)
	r.printInfo("\n" + graph)

	// Estimate timeline
	timeline := documents.EstimateProjectTimeline(mainTasks)
	r.printInfo(fmt.Sprintf("Estimated project timeline: %s", timeline))

	return fmt.Sprintf("Generated %d main tasks successfully", len(mainTasks)), nil
}

// generateSubTasks generates sub-tasks for a main task
func (r *REPL) generateSubTasks(ctx context.Context, taskID string) (string, error) {
	// Get main tasks from context
	r.session.mu.RLock()
	mainTasks, hasTasks := r.session.Context["main_tasks"].([]*documents.MainTask)
	currentPRD, _ := r.session.Context["current_prd"].(*documents.PRDEntity)
	currentTRD, _ := r.session.Context["current_trd"].(*documents.TRDEntity)
	r.session.mu.RUnlock()

	if !hasTasks {
		return "", fmt.Errorf("no main tasks found - generate main tasks first")
	}

	// Find the specified main task
	var targetTask *documents.MainTask
	for _, task := range mainTasks {
		if task.TaskID == taskID {
			targetTask = task
			break
		}
	}

	if targetTask == nil {
		return "", fmt.Errorf("main task not found: %s", taskID)
	}

	r.printInfo(fmt.Sprintf("Generating sub-tasks for %s: %s", taskID, targetTask.Name))

	// Generate sub-tasks
	subTasks, err := r.taskGen.GenerateSubTasks(targetTask, currentPRD, currentTRD)
	if err != nil {
		return "", fmt.Errorf("failed to generate sub-tasks: %w", err)
	}

	// Store in context
	r.session.mu.Lock()
	if r.session.Context["subtasks"] == nil {
		r.session.Context["subtasks"] = make(map[string][]*documents.SubTask)
	}
	subTaskMap := r.session.Context["subtasks"].(map[string][]*documents.SubTask)
	subTaskMap[taskID] = subTasks
	r.session.UpdatedAt = time.Now()
	r.session.mu.Unlock()

	// Display sub-task summary
	r.printInfo(fmt.Sprintf("Generated %d sub-tasks:", len(subTasks)))
	totalHours := 0
	for _, task := range subTasks {
		r.printInfo(fmt.Sprintf("  %s: %s (%d hours)", task.SubTaskID, task.Name, task.EstimatedHours))
		totalHours += task.EstimatedHours
	}
	r.printInfo(fmt.Sprintf("Total estimated hours: %d", totalHours))

	return fmt.Sprintf("Generated %d sub-tasks successfully", len(subTasks)), nil
}

// handleAnalyzeCommand handles document analysis
func (r *REPL) handleAnalyzeCommand(ctx context.Context, docType string, args []string) (string, error) {
	switch docType {
	case "prd":
		return r.analyzePRD(ctx)
	case "trd":
		return r.analyzeTRD(ctx)
	case "complexity":
		return r.analyzeComplexity(ctx)
	default:
		return "", fmt.Errorf("unsupported analysis type: %s", docType)
	}
}

// analyzePRD analyzes the current PRD
func (r *REPL) analyzePRD(ctx context.Context) (string, error) {
	r.session.mu.RLock()
	currentPRD, hasPRD := r.session.Context["current_prd"].(*documents.PRDEntity)
	r.session.mu.RUnlock()

	if !hasPRD {
		return "", fmt.Errorf("no PRD loaded")
	}

	analysis := fmt.Sprintf(`PRD Analysis:
  Title: %s
  Status: %s
  Complexity Score: %d/100
  Estimated Duration: %s
  Sections: %d
  Word Count: %d
  
Key Information:
  Project Name: %s
  Goals: %d identified
  Requirements: %d identified
  User Stories: %d identified
  Constraints: %d identified
  Keywords: %s`,
		currentPRD.Title,
		currentPRD.Status,
		currentPRD.ComplexityScore,
		currentPRD.EstimatedDuration,
		len(currentPRD.Sections),
		len(strings.Fields(currentPRD.Content)),
		currentPRD.ParsedContent.ProjectName,
		len(currentPRD.ParsedContent.Goals),
		len(currentPRD.ParsedContent.Requirements),
		len(currentPRD.ParsedContent.UserStories),
		len(currentPRD.ParsedContent.Constraints),
		strings.Join(currentPRD.ParsedContent.Keywords, ", "))

	return analysis, nil
}

// handleListCommand handles listing various items
func (r *REPL) handleListCommand(args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("specify what to list: tasks, documents, rules")
	}

	switch args[0] {
	case "tasks":
		return r.listTasks()
	case "documents":
		return r.listDocuments()
	case "rules":
		return r.listRules()
	default:
		return "", fmt.Errorf("unknown list type: %s", args[0])
	}
}

// listTasks lists all tasks in context
func (r *REPL) listTasks() (string, error) {
	r.session.mu.RLock()
	mainTasks, hasTasks := r.session.Context["main_tasks"].([]*documents.MainTask)
	r.session.mu.RUnlock()

	if !hasTasks || len(mainTasks) == 0 {
		return "No tasks found. Generate tasks first.", nil
	}

	var output strings.Builder
	output.WriteString("Main Tasks:\n")
	for _, task := range mainTasks {
		output.WriteString(fmt.Sprintf("  %s: %s\n", task.TaskID, task.Name))
		output.WriteString(fmt.Sprintf("    Phase: %s, Duration: %s, Complexity: %d\n",
			task.Phase, task.DurationEstimate, task.ComplexityScore))

		// Check for sub-tasks
		r.session.mu.RLock()
		if subTaskMap, ok := r.session.Context["subtasks"].(map[string][]*documents.SubTask); ok {
			if subTasks, hasSubTasks := subTaskMap[task.TaskID]; hasSubTasks {
				output.WriteString(fmt.Sprintf("    Sub-tasks: %d\n", len(subTasks)))
			}
		}
		r.session.mu.RUnlock()
	}

	return output.String(), nil
}

// listDocuments lists all documents in context
func (r *REPL) listDocuments() (string, error) {
	r.session.mu.RLock()
	defer r.session.mu.RUnlock()

	var output strings.Builder
	output.WriteString("Documents in session:\n")

	if prd, ok := r.session.Context["current_prd"].(*documents.PRDEntity); ok {
		output.WriteString(fmt.Sprintf("  PRD: %s (ID: %s)\n", prd.Title, prd.ID))
	}

	if trd, ok := r.session.Context["current_trd"].(*documents.TRDEntity); ok {
		output.WriteString(fmt.Sprintf("  TRD: %s (ID: %s)\n", trd.Title, trd.ID))
	}

	if mainTasks, ok := r.session.Context["main_tasks"].([]*documents.MainTask); ok {
		output.WriteString(fmt.Sprintf("  Main Tasks: %d tasks\n", len(mainTasks)))
	}

	if subTasks, ok := r.session.Context["subtasks"].(map[string][]*documents.SubTask); ok {
		totalSubTasks := 0
		for _, tasks := range subTasks {
			totalSubTasks += len(tasks)
		}
		output.WriteString(fmt.Sprintf("  Sub-tasks: %d total across %d main tasks\n", totalSubTasks, len(subTasks)))
	}

	return output.String(), nil
}

// listRules lists available generation rules
func (r *REPL) listRules() (string, error) {
	rules := r.ruleManager.ListRules()

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Available Rules (%d):\n", len(rules)))

	// Group by type
	byType := make(map[string][]map[string]interface{})
	for _, rule := range rules {
		ruleType := rule["type"].(documents.RuleType)
		byType[string(ruleType)] = append(byType[string(ruleType)], rule)
	}

	for ruleType, typeRules := range byType {
		output.WriteString(fmt.Sprintf("\n%s:\n", ruleType))
		for _, rule := range typeRules {
			active := "inactive"
			if rule["active"].(bool) {
				active = "active"
			}
			output.WriteString(fmt.Sprintf("  - %s (v%s, priority: %d, %s)\n",
				rule["name"],
				rule["version"],
				rule["priority"],
				active))
		}
	}

	return output.String(), nil
}

// handleShowCommand handles showing detailed information
func (r *REPL) handleShowCommand(item string, args []string) (string, error) {
	switch item {
	case "prd":
		return r.showPRD()
	case "trd":
		return r.showTRD()
	case "task":
		if len(args) == 0 {
			return "", fmt.Errorf("specify task ID")
		}
		return r.showTask(args[0])
	case "rule":
		if len(args) == 0 {
			return "", fmt.Errorf("specify rule name")
		}
		return r.showRule(args[0])
	default:
		return "", fmt.Errorf("unknown item type: %s", item)
	}
}

// showPRD shows current PRD content
func (r *REPL) showPRD() (string, error) {
	r.session.mu.RLock()
	currentPRD, hasPRD := r.session.Context["current_prd"].(*documents.PRDEntity)
	r.session.mu.RUnlock()

	if !hasPRD {
		return "", fmt.Errorf("no PRD loaded")
	}

	// Export to markdown format
	var output strings.Builder
	err := r.processor.ExportPRD(currentPRD, "markdown", &output)
	if err != nil {
		return "", fmt.Errorf("failed to export PRD: %w", err)
	}

	return output.String(), nil
}

// handleRulesCommand handles rule management
func (r *REPL) handleRulesCommand(args []string) error {
	if len(args) == 0 {
		rules := r.ruleManager.GetActiveRules()
		r.printInfo(fmt.Sprintf("Active rules: %d", len(rules)))
		for _, rule := range rules {
			r.printInfo(fmt.Sprintf("  - %s (%s)", rule.Name, rule.Type))
		}
		return nil
	}

	subcommand := args[0]
	switch subcommand {
	case "list":
		output, err := r.listRules()
		if err != nil {
			return err
		}
		r.printOutput(output)
		return nil

	case "show":
		if len(args) < 2 {
			return fmt.Errorf("specify rule name")
		}
		output, err := r.showRule(args[1])
		if err != nil {
			return err
		}
		r.printOutput(output)
		return nil

	case "edit":
		return fmt.Errorf("rule editing not yet implemented")

	default:
		return fmt.Errorf("unknown rules subcommand: %s", subcommand)
	}
}

// showRule shows detailed rule information
func (r *REPL) showRule(ruleName string) (string, error) {
	rule, err := r.ruleManager.GetRuleByName(ruleName)
	if err != nil {
		return "", err
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Rule: %s\n", rule.Name))
	output.WriteString(fmt.Sprintf("Type: %s\n", rule.Type))
	output.WriteString(fmt.Sprintf("Description: %s\n", rule.Description))
	output.WriteString(fmt.Sprintf("Version: %s\n", rule.Version))
	output.WriteString(fmt.Sprintf("Priority: %d\n", rule.Priority))
	output.WriteString(fmt.Sprintf("Active: %v\n", rule.Active))
	output.WriteString(fmt.Sprintf("\nContent:\n%s\n", rule.Content))

	return output.String(), nil
}

// createTRDInteractive creates a TRD interactively
func (r *REPL) createTRDInteractive(ctx context.Context) (string, error) {
	// Check if we have a current PRD
	r.session.mu.RLock()
	currentPRD, hasPRD := r.session.Context["current_prd"].(*documents.PRDEntity)
	r.session.mu.RUnlock()

	if !hasPRD {
		return "", fmt.Errorf("no PRD loaded - create or import a PRD first")
	}

	r.printInfo("Starting interactive TRD creation based on current PRD...")

	// Generate initial questions
	req := &ai.DocumentGenerationRequest{
		Type:        ai.DocumentTypeTRD,
		Interactive: true,
		Repository:  r.session.Repository,
		SessionID:   r.session.ID,
		SourcePRD:   currentPRD,
	}

	resp, err := r.documentGen.GenerateInteractive(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to start TRD creation: %w", err)
	}

	// Ask questions
	answers := []ai.InteractiveAnswer{}
	for _, question := range resp.Questions {
		answer, err := r.askQuestion(question)
		if err != nil {
			return "", err
		}
		answers = append(answers, answer)
	}

	// Continue with answers
	finalResp, err := r.documentGen.ContinueInteractive(ctx, r.session.ID, answers)
	if err != nil {
		return "", fmt.Errorf("failed to generate TRD: %w", err)
	}

	// Store in context
	r.session.mu.Lock()
	r.session.Context["current_trd"] = finalResp.Document
	r.session.UpdatedAt = time.Now()
	r.session.mu.Unlock()

	return fmt.Sprintf("TRD created successfully: %s", finalResp.Document.GetTitle()), nil
}

// createTasksInteractive creates tasks interactively
func (r *REPL) createTasksInteractive(ctx context.Context) (string, error) {
	// Check if we have PRD and TRD
	r.session.mu.RLock()
	_, hasPRD := r.session.Context["current_prd"].(*documents.PRDEntity)
	_, hasTRD := r.session.Context["current_trd"].(*documents.TRDEntity)
	r.session.mu.RUnlock()

	if !hasPRD || !hasTRD {
		return "", fmt.Errorf("both PRD and TRD are required for task creation")
	}

	// For now, just generate tasks automatically
	return r.generateTasks(ctx)
}

// showTask shows detailed task information
func (r *REPL) showTask(taskID string) (string, error) {
	r.session.mu.RLock()
	mainTasks, hasTasks := r.session.Context["main_tasks"].([]*documents.MainTask)
	r.session.mu.RUnlock()

	if !hasTasks {
		return "", fmt.Errorf("no tasks found")
	}

	// Find the task
	for _, task := range mainTasks {
		if task.TaskID == taskID {
			var output strings.Builder
			output.WriteString(fmt.Sprintf("Task: %s\n", task.TaskID))
			output.WriteString(fmt.Sprintf("Name: %s\n", task.Name))
			output.WriteString(fmt.Sprintf("Phase: %s\n", task.Phase))
			output.WriteString(fmt.Sprintf("Duration: %s\n", task.DurationEstimate))
			output.WriteString(fmt.Sprintf("Complexity: %d\n", task.ComplexityScore))
			output.WriteString(fmt.Sprintf("\nDescription:\n%s\n", task.Description))

			if len(task.Dependencies) > 0 {
				output.WriteString(fmt.Sprintf("\nDependencies: %s\n", strings.Join(task.Dependencies, ", ")))
			}

			if len(task.Deliverables) > 0 {
				output.WriteString("\nDeliverables:\n")
				for _, d := range task.Deliverables {
					output.WriteString(fmt.Sprintf("  - %s\n", d))
				}
			}

			if len(task.AcceptanceCriteria) > 0 {
				output.WriteString("\nAcceptance Criteria:\n")
				for _, ac := range task.AcceptanceCriteria {
					output.WriteString(fmt.Sprintf("  - %s\n", ac))
				}
			}

			// Check for sub-tasks
			r.session.mu.RLock()
			if subTaskMap, ok := r.session.Context["subtasks"].(map[string][]*documents.SubTask); ok {
				if subTasks, hasSubTasks := subTaskMap[taskID]; hasSubTasks {
					output.WriteString(fmt.Sprintf("\nSub-tasks (%d):\n", len(subTasks)))
					for _, st := range subTasks {
						output.WriteString(fmt.Sprintf("  %s: %s (%d hours)\n", st.SubTaskID, st.Name, st.EstimatedHours))
					}
				}
			}
			r.session.mu.RUnlock()

			return output.String(), nil
		}
	}

	return "", fmt.Errorf("task not found: %s", taskID)
}

// analyzeTRD analyzes the current TRD
func (r *REPL) analyzeTRD(ctx context.Context) (string, error) {
	r.session.mu.RLock()
	currentTRD, hasTRD := r.session.Context["current_trd"].(*documents.TRDEntity)
	r.session.mu.RUnlock()

	if !hasTRD {
		return "", fmt.Errorf("no TRD loaded")
	}

	analysis := fmt.Sprintf(`TRD Analysis:
  Title: %s
  Status: %s
  Sections: %d
  Word Count: %d
  
Technical Details:
  Architecture: %s
  Technology Stack: %s
  Dependencies: %d`,
		currentTRD.Title,
		currentTRD.Status,
		len(currentTRD.Sections),
		len(strings.Fields(currentTRD.Content)),
		currentTRD.Architecture,
		strings.Join(currentTRD.TechnicalStack, ", "),
		len(currentTRD.Dependencies))

	return analysis, nil
}

// analyzeComplexity performs complexity analysis
func (r *REPL) analyzeComplexity(ctx context.Context) (string, error) {
	r.session.mu.RLock()
	currentPRD, hasPRD := r.session.Context["current_prd"].(*documents.PRDEntity)
	currentTRD, hasTRD := r.session.Context["current_trd"].(*documents.TRDEntity)
	r.session.mu.RUnlock()

	if !hasPRD || !hasTRD {
		return "", fmt.Errorf("both PRD and TRD required for complexity analysis")
	}

	analyzer := documents.NewComplexityAnalyzer()
	analysis := analyzer.AnalyzeProject(currentPRD, currentTRD)

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Project Complexity Analysis:\n"))
	output.WriteString(fmt.Sprintf("  Total Complexity: %d/100\n", analysis.TotalComplexity))
	output.WriteString(fmt.Sprintf("  Project Type: %s\n", analysis.ProjectType))
	output.WriteString(fmt.Sprintf("  Requires Integration: %v\n", analysis.RequiresIntegration))

	if analysis.RequiresIntegration {
		output.WriteString(fmt.Sprintf("  Integration Complexity: %d\n", analysis.IntegrationComplexity))
	}

	output.WriteString(fmt.Sprintf("\nCore Features (%d):\n", len(analysis.CoreFeatures)))
	for _, feature := range analysis.CoreFeatures {
		output.WriteString(fmt.Sprintf("  - %s\n", feature))
	}

	if len(analysis.AdvancedFeatures) > 0 {
		output.WriteString(fmt.Sprintf("\nAdvanced Features (%d):\n", len(analysis.AdvancedFeatures)))
		for _, feature := range analysis.AdvancedFeatures {
			output.WriteString(fmt.Sprintf("  - %s\n", feature))
		}
	}

	return output.String(), nil
}

// showTRD shows current TRD content
func (r *REPL) showTRD() (string, error) {
	r.session.mu.RLock()
	currentTRD, hasTRD := r.session.Context["current_trd"].(*documents.TRDEntity)
	r.session.mu.RUnlock()

	if !hasTRD {
		return "", fmt.Errorf("no TRD loaded")
	}

	var output strings.Builder

	// Write title
	output.WriteString(fmt.Sprintf("# %s\n\n", currentTRD.Title))

	// Write metadata
	output.WriteString(fmt.Sprintf("**Status:** %s\n", currentTRD.Status))
	output.WriteString(fmt.Sprintf("**Architecture:** %s\n", currentTRD.Architecture))
	output.WriteString(fmt.Sprintf("**Tech Stack:** %s\n\n", strings.Join(currentTRD.TechnicalStack, ", ")))

	// Write sections
	for _, section := range currentTRD.Sections {
		prefix := strings.Repeat("#", section.Level)
		output.WriteString(fmt.Sprintf("%s %s\n\n", prefix, section.Title))
		output.WriteString(fmt.Sprintf("%s\n\n", section.Content))
	}

	return output.String(), nil
}

// Helper functions

func extractTechnicalStack(content string) []string {
	// Simple extraction based on common technology keywords
	stack := []string{}
	technologies := []string{
		"Go", "Python", "JavaScript", "TypeScript", "Java", "C++", "Rust",
		"React", "Vue", "Angular", "Node.js", "Django", "Flask", "Spring",
		"PostgreSQL", "MySQL", "MongoDB", "Redis", "Elasticsearch",
		"Docker", "Kubernetes", "AWS", "GCP", "Azure",
		"REST", "GraphQL", "gRPC", "WebSocket",
	}

	contentLower := strings.ToLower(content)
	for _, tech := range technologies {
		if strings.Contains(contentLower, strings.ToLower(tech)) {
			stack = append(stack, tech)
		}
	}

	return stack
}

func detectArchitecture(content string) string {
	patterns := map[string]string{
		"microservice":  "Microservices",
		"monolith":      "Monolithic",
		"serverless":    "Serverless",
		"event-driven":  "Event-Driven",
		"hexagonal":     "Hexagonal",
		"clean arch":    "Clean Architecture",
		"domain-driven": "Domain-Driven Design",
	}

	contentLower := strings.ToLower(content)
	for key, pattern := range patterns {
		if strings.Contains(contentLower, key) {
			return pattern
		}
	}

	return "Not Specified"
}

// Export functions for session data

// ExportSession exports the current session to a writer
func (r *REPL) ExportSession(writer io.Writer) error {
	r.session.mu.RLock()
	defer r.session.mu.RUnlock()

	sessionData := map[string]interface{}{
		"id":         r.session.ID,
		"mode":       r.session.Mode,
		"repository": r.session.Repository,
		"created_at": r.session.CreatedAt,
		"updated_at": r.session.UpdatedAt,
		"context":    r.session.Context,
		"history":    r.session.History,
	}

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(sessionData)
}

// ExportDocuments exports all documents to a directory
func (r *REPL) ExportDocuments(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	r.session.mu.RLock()
	defer r.session.mu.RUnlock()

	// Export PRD
	if prd, ok := r.session.Context["current_prd"].(*documents.PRDEntity); ok {
		prdFile := fmt.Sprintf("%s/prd_%s.md", dir, prd.ID)
		file, err := os.Create(prdFile)
		if err != nil {
			return fmt.Errorf("failed to create PRD file: %w", err)
		}
		defer file.Close()

		if err := r.processor.ExportPRD(prd, "markdown", file); err != nil {
			return fmt.Errorf("failed to export PRD: %w", err)
		}
	}

	// Export TRD
	if trd, ok := r.session.Context["current_trd"].(*documents.TRDEntity); ok {
		trdFile := fmt.Sprintf("%s/trd_%s.md", dir, trd.ID)
		file, err := os.Create(trdFile)
		if err != nil {
			return fmt.Errorf("failed to create TRD file: %w", err)
		}
		defer file.Close()

		// Export TRD (similar to PRD export)
		fmt.Fprintf(file, "# %s\n\n", trd.Title)
		fmt.Fprintf(file, "**Architecture:** %s\n", trd.Architecture)
		fmt.Fprintf(file, "**Tech Stack:** %s\n\n", strings.Join(trd.TechnicalStack, ", "))

		for _, section := range trd.Sections {
			prefix := strings.Repeat("#", section.Level)
			fmt.Fprintf(file, "%s %s\n\n%s\n\n", prefix, section.Title, section.Content)
		}
	}

	// Export tasks
	if mainTasks, ok := r.session.Context["main_tasks"].([]*documents.MainTask); ok {
		tasksFile := fmt.Sprintf("%s/main_tasks.json", dir)
		file, err := os.Create(tasksFile)
		if err != nil {
			return fmt.Errorf("failed to create tasks file: %w", err)
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(mainTasks); err != nil {
			return fmt.Errorf("failed to export tasks: %w", err)
		}
	}

	return nil
}
