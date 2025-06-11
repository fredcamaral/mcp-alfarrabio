#!/bin/bash

# Generate shell completion scripts for lmmc
# Usage: ./scripts/completions.sh

BINARY="./lmmc"
COMPLETIONS_DIR="completions"

# Ensure binary exists
if [ ! -f "$BINARY" ]; then
    echo "Error: Binary '$BINARY' not found. Please build first with 'go build -o lmmc ./cmd/lmmc'"
    exit 1
fi

# Create completions directory
mkdir -p "$COMPLETIONS_DIR"

echo "🔧 Generating shell completions..."

# Generate bash completion
echo "  Generating bash completion..."
$BINARY completion bash > "$COMPLETIONS_DIR/lmmc.bash"

# Generate zsh completion  
echo "  Generating zsh completion..."
$BINARY completion zsh > "$COMPLETIONS_DIR/lmmc.zsh"

# Generate fish completion
echo "  Generating fish completion..."
$BINARY completion fish > "$COMPLETIONS_DIR/lmmc.fish"

# Generate PowerShell completion
echo "  Generating PowerShell completion..."
$BINARY completion powershell > "$COMPLETIONS_DIR/lmmc.ps1"

echo "✅ Shell completions generated in $COMPLETIONS_DIR/"

# Show installation instructions
cat << 'EOF'

📋 Installation Instructions:

## Bash
echo 'source <(lmmc completion bash)' >>~/.bashrc
# Or for system-wide:
sudo cp completions/lmmc.bash /etc/bash_completion.d/lmmc

## Zsh  
echo 'source <(lmmc completion zsh)' >>~/.zshrc
# Or add to fpath:
mkdir -p ~/.local/share/zsh/site-functions
cp completions/lmmc.zsh ~/.local/share/zsh/site-functions/_lmmc

## Fish
cp completions/lmmc.fish ~/.config/fish/completions/

## PowerShell
# Add to PowerShell profile:
lmmc completion powershell | Out-String | Invoke-Expression

EOF