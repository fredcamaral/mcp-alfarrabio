# lmmc Homebrew Formula

This directory contains the Homebrew formula for installing lmmc (Lerian MCP Memory CLI).

## Distribution Process

### 1. Create a Homebrew Tap Repository

Create a separate repository named `homebrew-tap` under the `lerianstudio` organization:

```bash
# Create the tap repository
git clone https://github.com/lerianstudio/homebrew-tap.git
cd homebrew-tap

# Copy the formula
cp ../lerian-mcp-memory/homebrew-formula/lmmc.rb Formula/lmmc.rb

# Commit and push
git add Formula/lmmc.rb
git commit -m "Add lmmc formula"
git push origin main
```

### 2. Installation for Users

Once the tap repository is set up, users can install lmmc with:

```bash
# Add the tap
brew tap lerianstudio/tap

# Install lmmc
brew install lmmc

# Or in one command
brew install lerianstudio/tap/lmmc
```

### 3. Updating the Formula

When releasing new versions:

1. Update the `url` and `sha256` in the formula
2. Test the formula locally:
   ```bash
   brew install --build-from-source ./Formula/lmmc.rb
   brew test lmmc
   ```
3. Commit and push the updated formula

### 4. Formula Structure

The formula includes:

- **Description**: Clear description of lmmc's functionality
- **Dependencies**: Go build dependency
- **Installation**: Builds from source in the `cli` directory
- **Completions**: Automatic shell completion generation
- **Service**: Optional background service configuration
- **Tests**: Basic functionality verification

### 5. Release Automation

The GitHub Actions workflow in `.github/workflows/release.yml` automatically:

- Builds cross-platform binaries
- Calculates checksums
- Updates the formula with correct sha256
- Creates GitHub releases

### 6. Testing

To test the formula locally:

```bash
# Install from local file
brew install --build-from-source ./homebrew-formula/lmmc.rb

# Test functionality
brew test lmmc

# Verify installation
lmmc version
lmmc tui --help

# Uninstall
brew uninstall lmmc
```

### 7. Publishing Checklist

Before publishing a new version:

- [ ] Update version in formula
- [ ] Update URL to point to new release
- [ ] Calculate and update sha256 hash
- [ ] Test installation locally
- [ ] Verify all commands work
- [ ] Push to tap repository
- [ ] Announce release

## Formula Maintenance

The formula is designed to be:

- **Self-contained**: Builds from source with Go
- **Cross-platform**: Works on macOS and Linux
- **Feature-complete**: Includes completions and service configuration
- **Well-tested**: Comprehensive test suite

For questions or issues with the Homebrew formula, please open an issue in the main repository.