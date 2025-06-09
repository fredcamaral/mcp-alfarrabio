package repl

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"lerian-mcp-memory/internal/ai"
	"lerian-mcp-memory/internal/documents"
	"lerian-mcp-memory/internal/logging"

	"github.com/fatih/color"
	"github.com/google/uuid"
)

// Mode represents the REPL operation mode
type Mode string

const (
	ModeInteractive Mode = "interactive"
	ModeWorkflow    Mode = "workflow"
	ModeDebug       Mode = "debug"
)

// Session represents an interactive REPL session
type Session struct {
	ID               string
	Mode             Mode
	Context          map[string]interface{}
	History          []Command
	ActiveWorkflow   *Workflow
	HTTPServer       *HTTPServer
	NotificationChan chan Notification
	Repository       string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	mu               sync.RWMutex
}

// Command represents a command executed in the session
type Command struct {
	Input     string
	Output    string
	Error     error
	Timestamp time.Time
}

// Workflow represents an active document generation workflow
type Workflow struct {
	ID          string
	Type        string
	Stage       string
	Documents   map[string]documents.Document
	CurrentStep int
	TotalSteps  int
	StartTime   time.Time
}

// Notification represents a push notification
type Notification struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// REPL represents the Read-Eval-Print Loop interface
type REPL struct {
	session     *Session
	documentGen *ai.DocumentGenerator
	taskGen     *documents.TaskGenerator
	processor   *documents.Processor
	ruleManager *documents.RuleManager
	logger      logging.Logger
	input       io.Reader
	output      io.Writer
	colorOutput bool
	promptColor *color.Color
	outputColor *color.Color
	errorColor  *color.Color
	infoColor   *color.Color
}

// NewREPL creates a new REPL instance
func NewREPL(
	documentGen *ai.DocumentGenerator,
	taskGen *documents.TaskGenerator,
	processor *documents.Processor,
	ruleManager *documents.RuleManager,
	logger logging.Logger,
	repository string,
) *REPL {
	session := &Session{
		ID:               uuid.New().String(),
		Mode:             ModeInteractive,
		Context:          make(map[string]interface{}),
		History:          []Command{},
		NotificationChan: make(chan Notification, 100),
		Repository:       repository,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	return &REPL{
		session:     session,
		documentGen: documentGen,
		taskGen:     taskGen,
		processor:   processor,
		ruleManager: ruleManager,
		logger:      logger,
		input:       os.Stdin,
		output:      os.Stdout,
		colorOutput: true,
		promptColor: color.New(color.FgCyan, color.Bold),
		outputColor: color.New(color.FgGreen),
		errorColor:  color.New(color.FgRed),
		infoColor:   color.New(color.FgYellow),
	}
}

// Start starts the REPL session
func (r *REPL) Start(ctx context.Context, httpPort int) error {
	// Start HTTP server if port is specified
	if httpPort > 0 {
		r.session.HTTPServer = NewHTTPServer(r.session, httpPort, r.logger)
		go func() {
			if err := r.session.HTTPServer.Start(); err != nil && err != http.ErrServerClosed {
				r.logger.Error("HTTP server error", "error", err)
			}
		}()
		r.printInfo(fmt.Sprintf("HTTP server started on port %d for push notifications", httpPort))
	}

	// Print welcome message
	r.printWelcome()

	// Main REPL loop
	scanner := bufio.NewScanner(r.input)
	for {
		select {
		case <-ctx.Done():
			return r.shutdown()
		case notification := <-r.session.NotificationChan:
			r.handleNotification(notification)
		default:
			// Show prompt
			r.showPrompt()

			// Read input
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					return fmt.Errorf("input error: %w", err)
				}
				return r.shutdown()
			}

			input := strings.TrimSpace(scanner.Text())
			if input == "" {
				continue
			}

			// Process command
			if err := r.processCommand(ctx, input); err != nil {
				if err == io.EOF {
					return r.shutdown()
				}
				r.printError(fmt.Sprintf("Error: %v", err))
			}
		}
	}
}

// processCommand processes a single command
func (r *REPL) processCommand(ctx context.Context, input string) error {
	cmd := Command{
		Input:     input,
		Timestamp: time.Now(),
	}

	// Check for special commands
	if strings.HasPrefix(input, ":") {
		err := r.handleSpecialCommand(ctx, input)
		cmd.Error = err
		r.addToHistory(cmd)
		return err
	}

	// Process based on current mode
	switch r.session.Mode {
	case ModeInteractive:
		output, err := r.handleInteractiveCommand(ctx, input)
		cmd.Output = output
		cmd.Error = err
		r.addToHistory(cmd)
		if err != nil {
			return err
		}
		r.printOutput(output)

	case ModeWorkflow:
		output, err := r.handleWorkflowCommand(ctx, input)
		cmd.Output = output
		cmd.Error = err
		r.addToHistory(cmd)
		if err != nil {
			return err
		}
		r.printOutput(output)

	case ModeDebug:
		output, err := r.handleDebugCommand(ctx, input)
		cmd.Output = output
		cmd.Error = err
		r.addToHistory(cmd)
		if err != nil {
			return err
		}
		r.printOutput(output)
	}

	return nil
}

// handleSpecialCommand handles special REPL commands starting with ":"
func (r *REPL) handleSpecialCommand(ctx context.Context, input string) error {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	command := parts[0]
	args := parts[1:]

	switch command {
	case ":help", ":h":
		r.printHelp()
		return nil

	case ":quit", ":q", ":exit":
		return io.EOF

	case ":mode":
		if len(args) == 0 {
			r.printInfo(fmt.Sprintf("Current mode: %s", r.session.Mode))
			return nil
		}
		return r.setMode(Mode(args[0]))

	case ":context", ":ctx":
		r.printContext()
		return nil

	case ":history", ":hist":
		r.printHistory()
		return nil

	case ":clear":
		r.clearScreen()
		return nil

	case ":save":
		if len(args) == 0 {
			return fmt.Errorf("filename required")
		}
		return r.saveSession(args[0])

	case ":load":
		if len(args) == 0 {
			return fmt.Errorf("filename required")
		}
		return r.loadSession(args[0])

	case ":workflow":
		if len(args) == 0 {
			return r.showWorkflowStatus()
		}
		return r.handleWorkflowSubcommand(ctx, args)

	case ":rules":
		return r.handleRulesCommand(args)

	case ":status":
		r.printStatus()
		return nil

	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

// handleInteractiveCommand handles commands in interactive mode
func (r *REPL) handleInteractiveCommand(ctx context.Context, input string) (string, error) {
	// Parse command
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return "", nil
	}

	command := strings.ToLower(parts[0])
	args := parts[1:]

	switch command {
	case "create":
		if len(args) == 0 {
			return "", fmt.Errorf("specify what to create: prd, trd, tasks")
		}
		return r.handleCreateCommand(ctx, args[0], args[1:])

	case "import":
		if len(args) < 2 {
			return "", fmt.Errorf("usage: import <type> <file>")
		}
		return r.handleImportCommand(ctx, args[0], args[1])

	case "generate":
		if len(args) == 0 {
			return "", fmt.Errorf("specify what to generate")
		}
		return r.handleGenerateCommand(ctx, args[0], args[1:])

	case "analyze":
		if len(args) == 0 {
			return "", fmt.Errorf("specify what to analyze")
		}
		return r.handleAnalyzeCommand(ctx, args[0], args[1:])

	case "list":
		return r.handleListCommand(args)

	case "show":
		if len(args) == 0 {
			return "", fmt.Errorf("specify what to show")
		}
		return r.handleShowCommand(args[0], args[1:])

	default:
		// Treat as natural language for AI interaction
		return r.handleAIInteraction(ctx, input)
	}
}

// handleCreateCommand handles document creation
func (r *REPL) handleCreateCommand(ctx context.Context, docType string, args []string) (string, error) {
	switch docType {
	case "prd":
		return r.createPRDInteractive(ctx)
	case "trd":
		return r.createTRDInteractive(ctx)
	case "tasks":
		return r.createTasksInteractive(ctx)
	default:
		return "", fmt.Errorf("unknown document type: %s", docType)
	}
}

// createPRDInteractive creates a PRD interactively
func (r *REPL) createPRDInteractive(ctx context.Context) (string, error) {
	r.printInfo("Starting interactive PRD creation...")

	// Generate initial questions
	req := &ai.DocumentGenerationRequest{
		Type:        ai.DocumentTypePRD,
		Interactive: true,
		Repository:  r.session.Repository,
		SessionID:   r.session.ID,
	}

	resp, err := r.documentGen.GenerateInteractive(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to start PRD creation: %w", err)
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
		return "", fmt.Errorf("failed to generate PRD: %w", err)
	}

	// Store in context
	r.session.mu.Lock()
	r.session.Context["current_prd"] = finalResp.Document
	r.session.UpdatedAt = time.Now()
	r.session.mu.Unlock()

	// Send notification
	r.sendNotification(Notification{
		Type:    "document_created",
		Message: "PRD created successfully",
		Data: map[string]interface{}{
			"document_type": "prd",
			"document_id":   finalResp.Document.GetID(),
		},
	})

	return fmt.Sprintf("PRD created successfully: %s", finalResp.Document.GetTitle()), nil
}

// askQuestion prompts the user with a question
func (r *REPL) askQuestion(question ai.InteractiveQuestion) (ai.InteractiveAnswer, error) {
	r.printInfo(question.Question)

	if question.Type == "choice" && len(question.Options) > 0 {
		for i, option := range question.Options {
			fmt.Fprintf(r.output, "  %d. %s\n", i+1, option)
		}
		fmt.Fprintf(r.output, "Enter choice (1-%d): ", len(question.Options))
	} else {
		if question.Default != "" {
			fmt.Fprintf(r.output, "[%s]: ", question.Default)
		} else {
			fmt.Fprint(r.output, "> ")
		}
	}

	scanner := bufio.NewScanner(r.input)
	if !scanner.Scan() {
		return ai.InteractiveAnswer{}, fmt.Errorf("input cancelled")
	}

	answer := strings.TrimSpace(scanner.Text())

	// Use default if empty and available
	if answer == "" && question.Default != "" {
		answer = question.Default
	}

	// Validate choice
	if question.Type == "choice" && len(question.Options) > 0 {
		var choiceNum int
		if _, err := fmt.Sscanf(answer, "%d", &choiceNum); err == nil {
			if choiceNum >= 1 && choiceNum <= len(question.Options) {
				answer = question.Options[choiceNum-1]
			}
		}
	}

	return ai.InteractiveAnswer{
		QuestionID: question.ID,
		Answer:     answer,
	}, nil
}

// handleWorkflowCommand handles commands in workflow mode
func (r *REPL) handleWorkflowCommand(ctx context.Context, input string) (string, error) {
	if r.session.ActiveWorkflow == nil {
		return "", fmt.Errorf("no active workflow")
	}

	// Handle workflow-specific commands
	switch strings.ToLower(input) {
	case "next":
		return r.nextWorkflowStep(ctx)
	case "skip":
		return r.skipWorkflowStep(ctx)
	case "abort":
		return r.abortWorkflow()
	default:
		// Process as answer to current step
		return r.processWorkflowInput(ctx, input)
	}
}

// handleDebugCommand handles commands in debug mode
func (r *REPL) handleDebugCommand(ctx context.Context, input string) (string, error) {
	// In debug mode, show detailed information about command processing
	r.printInfo(fmt.Sprintf("Debug: Processing command: %s", input))

	// Show context state
	r.printInfo("Debug: Current context:")
	for k, v := range r.session.Context {
		r.printInfo(fmt.Sprintf("  %s: %v", k, v))
	}

	// Process command normally but with verbose output
	output, err := r.handleInteractiveCommand(ctx, input)

	if err != nil {
		r.printInfo(fmt.Sprintf("Debug: Error occurred: %v", err))
	}

	return output, err
}

// handleAIInteraction handles natural language AI interaction
func (r *REPL) handleAIInteraction(ctx context.Context, input string) (string, error) {
	// This would integrate with the AI service for natural language understanding
	return fmt.Sprintf("AI: I understand you want to: %s", input), nil
}

// Helper methods

func (r *REPL) printWelcome() {
	banner := `
╔═══════════════════════════════════════════════════════════════╗
║           Lerian MCP Memory - Interactive REPL                ║
║                                                               ║
║  AI-Powered Document Generation and Task Management           ║
╚═══════════════════════════════════════════════════════════════╝
`
	r.infoColor.Fprintln(r.output, banner)
	r.printInfo("Type :help for available commands")
}

func (r *REPL) printHelp() {
	help := `
Available Commands:

Document Commands:
  create prd          - Create a new PRD interactively
  create trd          - Create a TRD from existing PRD
  create tasks        - Generate tasks from PRD/TRD
  import <type> <file> - Import existing document
  generate <type>     - Generate document with AI
  analyze <doc>       - Analyze document complexity

Workflow Commands:
  :workflow start     - Start document generation workflow
  :workflow status    - Show workflow status
  :workflow abort     - Abort current workflow

REPL Commands:
  :help, :h          - Show this help
  :quit, :q          - Exit REPL
  :mode <mode>       - Change mode (interactive/workflow/debug)
  :context           - Show current context
  :history           - Show command history
  :clear             - Clear screen
  :save <file>       - Save session
  :load <file>       - Load session
  :rules             - Manage generation rules
  :status            - Show session status

In interactive mode, you can also use natural language to interact with AI.
`
	fmt.Fprint(r.output, help)
}

func (r *REPL) showPrompt() {
	prompt := fmt.Sprintf("[%s]> ", r.session.Mode)
	r.promptColor.Fprint(r.output, prompt)
}

func (r *REPL) printOutput(output string) {
	if r.colorOutput {
		r.outputColor.Fprintln(r.output, output)
	} else {
		fmt.Fprintln(r.output, output)
	}
}

func (r *REPL) printError(message string) {
	if r.colorOutput {
		r.errorColor.Fprintln(r.output, message)
	} else {
		fmt.Fprintln(r.output, message)
	}
}

func (r *REPL) printInfo(message string) {
	if r.colorOutput {
		r.infoColor.Fprintln(r.output, message)
	} else {
		fmt.Fprintln(r.output, message)
	}
}

func (r *REPL) printContext() {
	r.session.mu.RLock()
	defer r.session.mu.RUnlock()

	r.printInfo("Current Context:")
	for k, v := range r.session.Context {
		fmt.Fprintf(r.output, "  %s: %v\n", k, v)
	}
}

func (r *REPL) printHistory() {
	r.printInfo("Command History:")
	for i, cmd := range r.session.History {
		fmt.Fprintf(r.output, "%3d | %s | %s\n",
			i+1,
			cmd.Timestamp.Format("15:04:05"),
			cmd.Input)
	}
}

func (r *REPL) printStatus() {
	r.session.mu.RLock()
	defer r.session.mu.RUnlock()

	status := fmt.Sprintf(`
Session Status:
  ID:         %s
  Mode:       %s
  Repository: %s
  Created:    %s
  Duration:   %s
  Commands:   %d
`,
		r.session.ID,
		r.session.Mode,
		r.session.Repository,
		r.session.CreatedAt.Format("15:04:05"),
		time.Since(r.session.CreatedAt).Round(time.Second),
		len(r.session.History),
	)

	if r.session.ActiveWorkflow != nil {
		status += fmt.Sprintf(`
Active Workflow:
  Type:     %s
  Stage:    %s
  Progress: %d/%d
  Duration: %s
`,
			r.session.ActiveWorkflow.Type,
			r.session.ActiveWorkflow.Stage,
			r.session.ActiveWorkflow.CurrentStep,
			r.session.ActiveWorkflow.TotalSteps,
			time.Since(r.session.ActiveWorkflow.StartTime).Round(time.Second),
		)
	}

	if r.session.HTTPServer != nil {
		status += fmt.Sprintf("\nHTTP Server: Running on port %d\n", r.session.HTTPServer.port)
	}

	r.printInfo(status)
}

func (r *REPL) clearScreen() {
	// ANSI escape code to clear screen
	fmt.Fprint(r.output, "\033[2J\033[H")
}

func (r *REPL) setMode(mode Mode) error {
	validModes := []Mode{ModeInteractive, ModeWorkflow, ModeDebug}
	for _, m := range validModes {
		if m == mode {
			r.session.mu.Lock()
			r.session.Mode = mode
			r.session.UpdatedAt = time.Now()
			r.session.mu.Unlock()
			r.printInfo(fmt.Sprintf("Mode changed to: %s", mode))
			return nil
		}
	}
	return fmt.Errorf("invalid mode: %s", mode)
}

func (r *REPL) addToHistory(cmd Command) {
	r.session.mu.Lock()
	defer r.session.mu.Unlock()
	r.session.History = append(r.session.History, cmd)
	r.session.UpdatedAt = time.Now()
}

func (r *REPL) sendNotification(notification Notification) {
	notification.ID = uuid.New().String()
	notification.Timestamp = time.Now()

	select {
	case r.session.NotificationChan <- notification:
		// Sent successfully
	default:
		// Channel full, log warning
		r.logger.Warn("Notification channel full, dropping notification",
			"type", notification.Type)
	}
}

func (r *REPL) handleNotification(notification Notification) {
	// Display notification
	r.printInfo(fmt.Sprintf("\n[Notification] %s: %s", notification.Type, notification.Message))

	// Re-show prompt
	r.showPrompt()
}

func (r *REPL) saveSession(filename string) error {
	r.session.mu.RLock()
	defer r.session.mu.RUnlock()

	data, err := json.MarshalIndent(r.session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	r.printInfo(fmt.Sprintf("Session saved to: %s", filename))
	return nil
}

func (r *REPL) loadSession(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return fmt.Errorf("failed to unmarshal session: %w", err)
	}

	r.session.mu.Lock()
	defer r.session.mu.Unlock()

	// Update session fields (keep current ID and channels)
	r.session.Mode = session.Mode
	r.session.Context = session.Context
	r.session.History = session.History
	r.session.Repository = session.Repository
	r.session.UpdatedAt = time.Now()

	r.printInfo(fmt.Sprintf("Session loaded from: %s", filename))
	return nil
}

func (r *REPL) shutdown() error {
	r.printInfo("Shutting down...")

	// Stop HTTP server if running
	if r.session.HTTPServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := r.session.HTTPServer.Shutdown(ctx); err != nil {
			r.logger.Error("Error shutting down HTTP server", "error", err)
		}
	}

	return nil
}

// Workflow methods would be implemented here...
func (r *REPL) handleWorkflowSubcommand(ctx context.Context, args []string) error {
	// Implementation for workflow subcommands
	return fmt.Errorf("workflow commands not yet implemented")
}

func (r *REPL) showWorkflowStatus() error {
	// Implementation for showing workflow status
	return fmt.Errorf("workflow status not yet implemented")
}

func (r *REPL) nextWorkflowStep(ctx context.Context) (string, error) {
	// Implementation for next workflow step
	return "", fmt.Errorf("workflow steps not yet implemented")
}

func (r *REPL) skipWorkflowStep(ctx context.Context) (string, error) {
	// Implementation for skipping workflow step
	return "", fmt.Errorf("workflow skip not yet implemented")
}

func (r *REPL) abortWorkflow() (string, error) {
	// Implementation for aborting workflow
	return "", fmt.Errorf("workflow abort not yet implemented")
}

func (r *REPL) processWorkflowInput(ctx context.Context, input string) (string, error) {
	// Implementation for processing workflow input
	return "", fmt.Errorf("workflow input processing not yet implemented")
}

// Additional helper methods would be implemented here...
