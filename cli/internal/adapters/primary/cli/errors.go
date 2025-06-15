package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// CLIError represents an enhanced error with recovery suggestions
type CLIError struct {
	Code      string
	Message   string
	Details   string
	Recovery  []RecoveryOption
	SessionID string
}

// RecoveryOption provides a recovery suggestion
type RecoveryOption struct {
	Command     string
	Description string
}

// Error implements the error interface
func (e *CLIError) Error() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\n❌ Error: %s\n", e.Message))

	if e.Details != "" {
		sb.WriteString(fmt.Sprintf("   Details: %s\n", e.Details))
	}

	if len(e.Recovery) > 0 {
		sb.WriteString("\n💡 Recovery options:\n")
		for i, opt := range e.Recovery {
			sb.WriteString(fmt.Sprintf("   %d. %s\n", i+1, opt.Description))
			sb.WriteString(fmt.Sprintf("      Run: %s\n", opt.Command))
		}
	}

	if e.SessionID != "" {
		sb.WriteString("\n📌 Session saved. Use 'lmmc workflow status' to see progress.\n")
	}

	return sb.String()
}

// Common error constructors

// NewPRDNotFoundError creates an error for missing PRD
func NewPRDNotFoundError() *CLIError {
	return &CLIError{
		Code:    "PRD_NOT_FOUND",
		Message: "No PRD file found",
		Details: "Could not auto-detect a PRD file in the standard location",
		Recovery: []RecoveryOption{
			{
				Command:     "lmmc prd create \"your feature description\"",
				Description: "Create a new PRD interactively",
			},
			{
				Command:     "lmmc prd import /path/to/existing-prd.md",
				Description: "Import an existing PRD file",
			},
			{
				Command:     "lmmc trd create --from-prd /path/to/prd.md",
				Description: "Specify PRD path explicitly",
			},
		},
	}
}

// NewTRDGenerationError creates an error for TRD generation failures
func NewTRDGenerationError(reason string) *CLIError {
	return &CLIError{
		Code:    "TRD_GENERATION_FAILED",
		Message: "TRD generation failed at technical analysis phase",
		Details: reason,
		Recovery: []RecoveryOption{
			{
				Command:     "lmmc prd enhance --add-technical-details",
				Description: "Add more technical details to the PRD",
			},
			{
				Command:     "lmmc trd create --skip-architecture",
				Description: "Skip architecture analysis (not recommended)",
			},
			{
				Command:     "lmmc workflow continue --from trd",
				Description: "Resume workflow from TRD generation",
			},
		},
	}
}

// NewTaskValidationError creates an error for task validation failures
func NewTaskValidationError(taskID string, issues []string) *CLIError {
	details := fmt.Sprintf("Task %s has the following issues:\n", taskID)
	for _, issue := range issues {
		details += fmt.Sprintf("      - %s\n", issue)
	}

	return &CLIError{
		Code:    "TASK_VALIDATION_FAILED",
		Message: fmt.Sprintf("Task %s is not atomic", taskID),
		Details: details,
		Recovery: []RecoveryOption{
			{
				Command:     "lmmc tasks split " + taskID,
				Description: "Split this task into smaller atomic tasks",
			},
			{
				Command:     fmt.Sprintf("lmmc tasks edit %s --add-deliverables", taskID),
				Description: "Add clear deliverables to the task",
			},
			{
				Command:     "lmmc tasks regenerate --atomic",
				Description: "Regenerate all tasks with atomic validation",
			},
		},
	}
}

// NewAIServiceError creates an error for AI service failures
func NewAIServiceError(provider string) *CLIError {
	return &CLIError{
		Code:    "AI_SERVICE_UNAVAILABLE",
		Message: fmt.Sprintf("AI service '%s' is not available", provider),
		Details: "Check your API key and network connection",
		Recovery: []RecoveryOption{
			{
				Command:     "lmmc config set ai.api_key YOUR_API_KEY",
				Description: "Set your AI provider API key",
			},
			{
				Command:     "lmmc ai test-connection",
				Description: "Test AI service connectivity",
			},
			{
				Command:     "lmmc prd create --ai-provider local",
				Description: "Use a local AI model instead",
			},
		},
	}
}

// NewWorkflowInterruptedError creates an error for interrupted workflows
func NewWorkflowInterruptedError(step string) *CLIError {
	return &CLIError{
		Code:    "WORKFLOW_INTERRUPTED",
		Message: "Workflow interrupted at step: " + step,
		Details: "Your progress has been saved",
		Recovery: []RecoveryOption{
			{
				Command:     "lmmc workflow continue",
				Description: "Resume from where you left off",
			},
			{
				Command:     "lmmc workflow status",
				Description: "Check current workflow status",
			},
			{
				Command:     "lmmc workflow restart --from " + step,
				Description: "Restart from this specific step",
			},
		},
		SessionID: "current",
	}
}

// markFlagRequired is a helper to mark flags as required and handle errors
func markFlagRequired(cmd *cobra.Command, flags ...string) {
	for _, flag := range flags {
		if err := cmd.MarkFlagRequired(flag); err != nil {
			// Log the error but don't fail - this is a setup issue
			// that should be caught during development
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to mark flag '%s' as required: %v\n", flag, err)
		}
	}
}

// markFlagHidden is a helper to mark flags as hidden and handle errors
func markFlagHidden(cmd *cobra.Command, flags ...string) {
	for _, flag := range flags {
		if err := cmd.Flags().MarkHidden(flag); err != nil {
			// Log the error but don't fail - this is a setup issue
			// that should be caught during development
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to mark flag '%s' as hidden: %v\n", flag, err)
		}
	}
}

// WrapError wraps a standard error with CLI enhancements
func WrapError(err error, code string) *CLIError {
	if err == nil {
		return nil
	}

	// Check if it's already a CLIError
	if cliErr, ok := err.(*CLIError); ok {
		return cliErr
	}

	// Create enhanced error based on error message
	msg := err.Error()

	// Pattern matching for common errors
	switch {
	case strings.Contains(msg, "no PRD"):
		return NewPRDNotFoundError()
	case strings.Contains(msg, "AI service"):
		return NewAIServiceError("default")
	case strings.Contains(msg, "validation"):
		return &CLIError{
			Code:    code,
			Message: msg,
			Recovery: []RecoveryOption{
				{
					Command:     "lmmc tasks validate --fix",
					Description: "Attempt automatic fixes",
				},
			},
		}
	default:
		return &CLIError{
			Code:    code,
			Message: msg,
			Details: "An unexpected error occurred",
			Recovery: []RecoveryOption{
				{
					Command:     "lmmc --verbose [command]",
					Description: "Run with verbose logging for more details",
				},
				{
					Command:     "lmmc help",
					Description: "Show available commands and options",
				},
			},
		}
	}
}
