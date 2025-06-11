package cli

import (
	"os"

	"github.com/spf13/cobra"
)

// createCompletionCommand creates the 'completion' command
func (c *CLI) createCompletionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for lmmc.

The completion script for each shell will be written to stdout.

To install completions:

Bash:
  $ echo 'source <(lmmc completion bash)' >>~/.bashrc

Zsh:
  $ echo 'source <(lmmc completion zsh)' >>~/.zshrc

Fish:
  $ lmmc completion fish > ~/.config/fish/completions/lmmc.fish

PowerShell:
  $ lmmc completion powershell | Out-String | Invoke-Expression`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return c.RootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				return c.RootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return c.RootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return c.RootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return cmd.Help()
			}
		},
	}

	return cmd
}
