// Package repl provides an interactive Read-Eval-Print Loop for PRD/TRD document processing.
package repl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
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
	case DocTypePRD:
		return r.importPRD(ctx, filename)
	case DocTypeTRD:
		return r.importTRD(ctx, filename)
	default:
		return "", fmt.Errorf("unsupported document type for import: %s", docType)
	}
}

// importPRD imports a PRD from file
func (r *REPL) importPRD(ctx context.Context, filename string) (string, error) {
	_ = ctx // unused parameter, kept for potential future context-aware operations
	r.printInfo("Importing PRD from: " + filename)

	// Process the PRD file
	prd, err := r.processor.ProcessPRDFile(filename, r.session.Repository)
	if err != nil {
		return "", fmt.Errorf("failed to process PRD: %w", err)
	}

	// Validate PRD
	if err := r.processor.ValidatePRD(prd); err != nil {
		r.printInfo("Warning: PRD validation issues: " + err.Error())
	}

	// Store in context
	r.session.mu.Lock()
	r.session.Context["current_prd"] = prd
	r.session.Context["prd_filename"] = filename
	r.session.UpdatedAt = time.Now()
	r.session.mu.Unlock()

	// Send notification
	r.sendNotification(&Notification{
		Type:    "document_imported",
		Message: "PRD imported: " + prd.Title,
		Data: map[string]interface{}{
			"document_type": DocTypePRD,
			"document_id":   prd.ID,
			"complexity":    prd.ComplexityScore,
		},
	})

	return "PRD imported successfully: " + prd.Title + " (Complexity: " + strconv.Itoa(prd.ComplexityScore) + "/100)", nil
}

// importTRD imports a TRD from file
func (r *REPL) importTRD(ctx context.Context, filename string) (string, error) {
	_ = ctx // unused parameter, kept for potential future context-aware operations
	// Check if we have a current PRD
	r.session.mu.RLock()
	currentPRD, hasPRD := r.session.Context["current_prd"].(*documents.PRDEntity)
	r.session.mu.RUnlock()

	if !hasPRD {
		return "", errors.New("no PRD loaded - import or create a PRD first")
	}

	r.printInfo("Importing TRD from: " + filename)

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

	return "TRD imported successfully: " + trd.Title, nil
}

// handleGenerateCommand handles document generation
func (r *REPL) handleGenerateCommand(ctx context.Context, docType string, args []string) (string, error) {
	switch docType {
	case DocTypeTRD:
		return r.generateTRD(ctx)
	case "tasks":
		return r.generateTasks(ctx)
	case "subtasks":
		if len(args) == 0 {
			return "", errors.New("specify main task ID")
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
		return "", errors.New("no PRD loaded - import or create a PRD first")
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
			"prd_complexity": strconv.Itoa(currentPRD.ComplexityScore),
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
			r.printInfo("  - " + suggestion)
		}
	}

	return "TRD generated successfully (Model: " + string(resp.ModelUsed) + ", Tokens: " + strconv.Itoa(resp.TokensUsed.Total) + ")", nil
}

// generateTasks generates main tasks from PRD and TRD
func (r *REPL) generateTasks(ctx context.Context) (string, error) {
	_ = ctx // unused parameter, kept for potential future context-aware operations
	// Check if we have PRD and TRD
	r.session.mu.RLock()
	currentPRD, hasPRD := r.session.Context["current_prd"].(*documents.PRDEntity)
	currentTRD, hasTRD := r.session.Context["current_trd"].(*documents.TRDEntity)
	r.session.mu.RUnlock()

	if !hasPRD || !hasTRD {
		return "", errors.New("both PRD and TRD are required for task generation")
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
	r.printInfo("Generated " + strconv.Itoa(len(mainTasks)) + " main tasks:")
	for _, task := range mainTasks {
		r.printInfo("  " + task.TaskID + ": " + task.Name + " (" + task.DurationEstimate + ")")
	}

	// Generate dependency graph
	graph := documents.GenerateTaskDependencyGraph(mainTasks)
	r.printInfo("\n" + graph)

	// Estimate timeline
	timeline := documents.EstimateProjectTimeline(mainTasks)
	r.printInfo("Estimated project timeline: " + timeline)

	return "Generated " + strconv.Itoa(len(mainTasks)) + " main tasks successfully", nil
}

// generateSubTasks generates sub-tasks for a main task
func (r *REPL) generateSubTasks(ctx context.Context, taskID string) (string, error) {
	_ = ctx // unused parameter, kept for potential future context-aware operations
	// Get main tasks from context
	r.session.mu.RLock()
	mainTasks, hasTasks := r.session.Context["main_tasks"].([]*documents.MainTask)
	currentPRD, _ := r.session.Context["current_prd"].(*documents.PRDEntity)
	currentTRD, _ := r.session.Context["current_trd"].(*documents.TRDEntity)
	r.session.mu.RUnlock()

	if !hasTasks {
		return "", errors.New("no main tasks found - generate main tasks first")
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

	r.printInfo("Generating sub-tasks for " + taskID + ": " + targetTask.Name)

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
	r.printInfo("Generated " + strconv.Itoa(len(subTasks)) + " sub-tasks:")
	totalHours := 0
	for _, task := range subTasks {
		r.printInfo("  " + task.SubTaskID + ": " + task.Name + " (" + strconv.Itoa(task.EstimatedHours) + " hours)")
		totalHours += task.EstimatedHours
	}
	r.printInfo("Total estimated hours: " + strconv.Itoa(totalHours))

	return "Generated " + strconv.Itoa(len(subTasks)) + " sub-tasks successfully", nil
}

// handleAnalyzeCommand handles document analysis
func (r *REPL) handleAnalyzeCommand(ctx context.Context, docType string, args []string) (string, error) {
	_ = args // unused parameter, kept for potential future argument-based analysis
	switch docType {
	case DocTypePRD:
		return r.analyzePRD(ctx)
	case DocTypeTRD:
		return r.analyzeTRD(ctx)
	case "complexity":
		return r.analyzeComplexity(ctx)
	default:
		return "", fmt.Errorf("unsupported analysis type: %s", docType)
	}
}

// analyzePRD analyzes the current PRD
func (r *REPL) analyzePRD(ctx context.Context) (string, error) {
	_ = ctx // unused parameter, kept for potential future context-aware operations
	r.session.mu.RLock()
	currentPRD, hasPRD := r.session.Context["current_prd"].(*documents.PRDEntity)
	r.session.mu.RUnlock()

	if !hasPRD {
		return "", errors.New("no PRD loaded")
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
		return "", errors.New("specify what to list: tasks, documents, rules")
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
		output.WriteString("  " + task.TaskID + ": " + task.Name + "\n")
		output.WriteString("    Phase: " + task.Phase + ", Duration: " + task.DurationEstimate + ", Complexity: " + strconv.Itoa(task.ComplexityScore) + "\n")

		// Check for sub-tasks
		r.session.mu.RLock()
		if subTaskMap, ok := r.session.Context["subtasks"].(map[string][]*documents.SubTask); ok {
			if subTasks, hasSubTasks := subTaskMap[task.TaskID]; hasSubTasks {
				output.WriteString("    Sub-tasks: " + strconv.Itoa(len(subTasks)) + "\n")
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
		output.WriteString("  PRD: " + prd.Title + " (ID: " + prd.ID + ")\n")
	}

	if trd, ok := r.session.Context["current_trd"].(*documents.TRDEntity); ok {
		output.WriteString("  TRD: " + trd.Title + " (ID: " + trd.ID + ")\n")
	}

	if mainTasks, ok := r.session.Context["main_tasks"].([]*documents.MainTask); ok {
		output.WriteString("  Main Tasks: " + strconv.Itoa(len(mainTasks)) + " tasks\n")
	}

	if subTasks, ok := r.session.Context["subtasks"].(map[string][]*documents.SubTask); ok {
		totalSubTasks := 0
		for _, tasks := range subTasks {
			totalSubTasks += len(tasks)
		}
		output.WriteString("  Sub-tasks: " + strconv.Itoa(totalSubTasks) + " total across " + strconv.Itoa(len(subTasks)) + " main tasks\n")
	}

	return output.String(), nil
}

// listRules lists available generation rules
func (r *REPL) listRules() (string, error) {
	rules := r.ruleManager.ListRules()

	var output strings.Builder
	output.WriteString("Available Rules (" + strconv.Itoa(len(rules)) + "):\n")

	// Group by type
	byType := make(map[string][]map[string]interface{})
	for _, rule := range rules {
		ruleType := rule["type"].(documents.RuleType)
		byType[string(ruleType)] = append(byType[string(ruleType)], rule)
	}

	for ruleType, typeRules := range byType {
		output.WriteString("\n" + ruleType + ":\n")
		for _, rule := range typeRules {
			active := "inactive"
			if rule["active"].(bool) {
				active = "active"
			}
			output.WriteString("  - " + rule["name"].(string) + " (v" + rule["version"].(string) + ", priority: " + strconv.Itoa(rule["priority"].(int)) + ", " + active + ")\n")
		}
	}

	return output.String(), nil
}

// handleShowCommand handles showing detailed information
func (r *REPL) handleShowCommand(item string, args []string) (string, error) {
	switch item {
	case DocTypePRD:
		return r.showPRD()
	case DocTypeTRD:
		return r.showTRD()
	case "task":
		if len(args) == 0 {
			return "", errors.New("specify task ID")
		}
		return r.showTask(args[0])
	case "rule":
		if len(args) == 0 {
			return "", errors.New("specify rule name")
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
		return "", errors.New("no PRD loaded")
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
func (r *REPL) handleRulesCommand(_ context.Context, args []string) error {
	if len(args) == 0 {
		rules := r.ruleManager.GetActiveRules()
		r.printInfo("Active rules: " + strconv.Itoa(len(rules)))
		for _, rule := range rules {
			r.printInfo("  - " + rule.Name + " (" + string(rule.Type) + ")")
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
			return errors.New("specify rule name")
		}
		output, err := r.showRule(args[1])
		if err != nil {
			return err
		}
		r.printOutput(output)
		return nil

	case "edit":
		return errors.New("rule editing not yet implemented")

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
	output.WriteString("Rule: " + rule.Name + "\n")
	output.WriteString("Type: " + string(rule.Type) + "\n")
	output.WriteString("Description: " + rule.Description + "\n")
	output.WriteString("Version: " + rule.Version + "\n")
	output.WriteString("Priority: " + strconv.Itoa(rule.Priority) + "\n")
	output.WriteString("Active: " + strconv.FormatBool(rule.Active) + "\n")
	output.WriteString("\nContent:\n" + rule.Content + "\n")

	return output.String(), nil
}

// createTRDInteractive creates a TRD interactively
func (r *REPL) createTRDInteractive(ctx context.Context) (string, error) {
	// Check if we have a current PRD
	r.session.mu.RLock()
	currentPRD, hasPRD := r.session.Context["current_prd"].(*documents.PRDEntity)
	r.session.mu.RUnlock()

	if !hasPRD {
		return "", errors.New("no PRD loaded - create or import a PRD first")
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
		answer, err := r.askQuestion(&question)
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

	return "TRD created successfully: " + finalResp.Document.GetTitle(), nil
}

// createTasksInteractive creates tasks interactively
func (r *REPL) createTasksInteractive(ctx context.Context) (string, error) {
	// Check if we have PRD and TRD
	r.session.mu.RLock()
	_, hasPRD := r.session.Context["current_prd"].(*documents.PRDEntity)
	_, hasTRD := r.session.Context["current_trd"].(*documents.TRDEntity)
	r.session.mu.RUnlock()

	if !hasPRD || !hasTRD {
		return "", errors.New("both PRD and TRD are required for task creation")
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
		return "", errors.New("no tasks found")
	}

	// Find the task
	for _, task := range mainTasks {
		if task.TaskID != taskID {
			continue
		}

		var output strings.Builder
		output.WriteString("Task: " + task.TaskID + "\n")
		output.WriteString("Name: " + task.Name + "\n")
		output.WriteString("Phase: " + task.Phase + "\n")
		output.WriteString("Duration: " + task.DurationEstimate + "\n")
		output.WriteString("Complexity: " + strconv.Itoa(task.ComplexityScore) + "\n")
		output.WriteString("\nDescription:\n" + task.Description + "\n")

		if len(task.Dependencies) > 0 {
			output.WriteString("\nDependencies: " + strings.Join(task.Dependencies, ", ") + "\n")
		}

		if len(task.Deliverables) > 0 {
			output.WriteString("\nDeliverables:\n")
			for _, d := range task.Deliverables {
				output.WriteString("  - " + d + "\n")
			}
		}

		if len(task.AcceptanceCriteria) > 0 {
			output.WriteString("\nAcceptance Criteria:\n")
			for _, ac := range task.AcceptanceCriteria {
				output.WriteString("  - " + ac + "\n")
			}
		}

		// Check for sub-tasks
		r.session.mu.RLock()
		if subTaskMap, ok := r.session.Context["subtasks"].(map[string][]*documents.SubTask); ok {
			if subTasks, hasSubTasks := subTaskMap[taskID]; hasSubTasks {
				output.WriteString("\nSub-tasks (" + strconv.Itoa(len(subTasks)) + "):\n")
				for _, st := range subTasks {
					output.WriteString("  " + st.SubTaskID + ": " + st.Name + " (" + strconv.Itoa(st.EstimatedHours) + " hours)\n")
				}
			}
		}
		r.session.mu.RUnlock()

		return output.String(), nil
	}

	return "", fmt.Errorf("task not found: %s", taskID)
}

// analyzeTRD analyzes the current TRD
func (r *REPL) analyzeTRD(ctx context.Context) (string, error) {
	_ = ctx // unused parameter, kept for potential future context-aware operations
	r.session.mu.RLock()
	currentTRD, hasTRD := r.session.Context["current_trd"].(*documents.TRDEntity)
	r.session.mu.RUnlock()

	if !hasTRD {
		return "", errors.New("no TRD loaded")
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
	_ = ctx // unused parameter, kept for potential future context-aware operations
	r.session.mu.RLock()
	currentPRD, hasPRD := r.session.Context["current_prd"].(*documents.PRDEntity)
	currentTRD, hasTRD := r.session.Context["current_trd"].(*documents.TRDEntity)
	r.session.mu.RUnlock()

	if !hasPRD || !hasTRD {
		return "", errors.New("both PRD and TRD required for complexity analysis")
	}

	analyzer := documents.NewComplexityAnalyzer()
	analysis := analyzer.AnalyzeProject(currentPRD, currentTRD)

	var output strings.Builder
	output.WriteString("Project Complexity Analysis:\n")
	output.WriteString("  Total Complexity: " + strconv.Itoa(analysis.TotalComplexity) + "/100\n")
	output.WriteString("  Project Type: " + analysis.ProjectType + "\n")
	output.WriteString("  Requires Integration: " + strconv.FormatBool(analysis.RequiresIntegration) + "\n")

	if analysis.RequiresIntegration {
		output.WriteString("  Integration Complexity: " + strconv.Itoa(analysis.IntegrationComplexity) + "\n")
	}

	output.WriteString("\nCore Features (" + strconv.Itoa(len(analysis.CoreFeatures)) + "):\n")
	for _, feature := range analysis.CoreFeatures {
		output.WriteString("  - " + feature + "\n")
	}

	if len(analysis.AdvancedFeatures) > 0 {
		output.WriteString("\nAdvanced Features (" + strconv.Itoa(len(analysis.AdvancedFeatures)) + "):\n")
		for _, feature := range analysis.AdvancedFeatures {
			output.WriteString("  - " + feature + "\n")
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
		return "", errors.New("no TRD loaded")
	}

	var output strings.Builder

	// Write title
	output.WriteString("# " + currentTRD.Title + "\n\n")

	// Write metadata
	output.WriteString("**Status:** " + string(currentTRD.Status) + "\n")
	output.WriteString("**Architecture:** " + currentTRD.Architecture + "\n")
	output.WriteString("**Tech Stack:** " + strings.Join(currentTRD.TechnicalStack, ", ") + "\n\n")

	// Write sections
	for _, section := range currentTRD.Sections {
		prefix := strings.Repeat("#", section.Level)
		output.WriteString(prefix + " " + section.Title + "\n\n")
		output.WriteString(section.Content + "\n\n")
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
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	r.session.mu.RLock()
	defer r.session.mu.RUnlock()

	if err := r.exportPRD(dir); err != nil {
		return err
	}

	if err := r.exportTRD(dir); err != nil {
		return err
	}

	if err := r.exportTasks(dir); err != nil {
		return err
	}

	return nil
}

// exportPRD exports PRD document if it exists
func (r *REPL) exportPRD(dir string) error {
	prd, ok := r.session.Context["current_prd"].(*documents.PRDEntity)
	if !ok {
		return nil
	}

	prdFile := dir + "/prd_" + prd.ID + ".md"
	file, err := os.Create(prdFile)
	if err != nil {
		return fmt.Errorf("failed to create PRD file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Failed to close file: %v", err)
		}
	}()

	return r.processor.ExportPRD(prd, "markdown", file)
}

// exportTRD exports TRD document if it exists
func (r *REPL) exportTRD(dir string) error {
	trd, ok := r.session.Context["current_trd"].(*documents.TRDEntity)
	if !ok {
		return nil
	}

	trdFile := dir + "/trd_" + trd.ID + ".md"
	file, err := os.Create(trdFile)
	if err != nil {
		return fmt.Errorf("failed to create TRD file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Failed to close file: %v", err)
		}
	}()

	return r.writeTRDContent(file, trd)
}

// writeTRDContent writes TRD content to file
func (r *REPL) writeTRDContent(file *os.File, trd *documents.TRDEntity) error {
	if _, err := fmt.Fprintf(file, "# %s\n\n", trd.Title); err != nil {
		return fmt.Errorf("failed to write TRD title: %w", err)
	}
	if _, err := fmt.Fprintf(file, "**Architecture:** %s\n", trd.Architecture); err != nil {
		return fmt.Errorf("failed to write TRD architecture: %w", err)
	}
	if _, err := fmt.Fprintf(file, "**Tech Stack:** %s\n\n", strings.Join(trd.TechnicalStack, ", ")); err != nil {
		return fmt.Errorf("failed to write TRD tech stack: %w", err)
	}

	for _, section := range trd.Sections {
		prefix := strings.Repeat("#", section.Level)
		if _, err := fmt.Fprintf(file, "%s %s\n\n%s\n\n", prefix, section.Title, section.Content); err != nil {
			return fmt.Errorf("failed to write TRD section: %w", err)
		}
	}
	return nil
}

// exportTasks exports tasks if they exist
func (r *REPL) exportTasks(dir string) error {
	mainTasks, ok := r.session.Context["main_tasks"].([]*documents.MainTask)
	if !ok {
		return nil
	}

	tasksFile := dir + "/main_tasks.json"
	file, err := os.Create(tasksFile)
	if err != nil {
		return fmt.Errorf("failed to create tasks file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Failed to close file: %v", err)
		}
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(mainTasks)
}
