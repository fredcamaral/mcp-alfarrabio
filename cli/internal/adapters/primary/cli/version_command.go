package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Build information variables (set by ldflags during build)
var (
	Version   = "dev"     // Version is set during build
	BuildTime = "unknown" // BuildTime is set during build
	GitCommit = "unknown" // GitCommit is set during build
)

// createVersionCommand creates the 'version' command
func (c *CLI) createVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  "Display version, build time, and system information for lmmc",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if detailed output is requested
			detailed, _ := cmd.Flags().GetBool("detailed")

			if detailed {
				// Detailed version information
				fmt.Printf("lmmc version %s\n", Version)
				fmt.Printf("Build time:  %s\n", BuildTime)
				fmt.Printf("Git commit:  %s\n", GitCommit)
				fmt.Printf("Go version:  %s\n", runtime.Version())
				fmt.Printf("OS/Arch:     %s/%s\n", runtime.GOOS, runtime.GOARCH)
				fmt.Printf("Compiler:    %s\n", runtime.Compiler)
			} else {
				// Simple version output
				fmt.Printf("lmmc version %s\n", Version)
			}

			return nil
		},
	}

	cmd.Flags().BoolP("detailed", "d", false, "Show detailed version information")

	return cmd
}
