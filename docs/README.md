# MCP Memory Server Documentation

Welcome to the MCP Memory Server documentation! This directory contains all technical documentation, guides, and references for the project.

## 📚 Documentation Index

### 🚀 Getting Started
- [Project Overview](../README.md) - Main project README with features and quick start
- [Development Setup](DEV-HOT-RELOAD.md) - Hot reload development environment setup
- [Deployment Guide](DEPLOYMENT.md) - Production deployment instructions
- [GraphQL API & Web UI](GRAPHQL_WEB_UI.md) - GraphQL server and web interface guide
- [ChromaDB Persistence Fix](CHROMADB_PERSISTENCE_FIX.md) - ChromaDB configuration for data persistence

### 🛠️ Development
- [Development Roadmap](ROADMAP.md) - Current priorities and future improvements
- [Architecture Overview](ARCHITECTURE.md) - System design and component interaction *(coming soon)*
- [API Reference](website/api-reference.md) - Complete API documentation
- [Examples](website/examples.md) - Code examples and usage patterns

### 🔧 Operations
- [Monitoring Setup](MONITORING.md) - Prometheus, Grafana, and observability
- [Troubleshooting Guide](TROUBLESHOOTING.md) - Common issues and solutions *(coming soon)*
- [Configuration Reference](CONFIGURATION.md) - All configuration options *(coming soon)*

### 🌐 Website Content
- [Homepage](website/index.md) - Website landing page content
- [Getting Started Guide](website/getting-started.md) - User-friendly introduction
- [FAQ](website/faq.md) - Frequently asked questions

### 🔌 IDE Integration
- [VS Code Extension](ide/vscode-extension.md) - Visual Studio Code integration
- [IntelliJ Plugin](ide/intellij-plugin.md) - JetBrains IDE integration

### 📣 Marketing Materials
- [Launch Announcement](marketing/launch-announcement.md) - Public launch messaging
- [Feature Highlights](marketing/feature-highlights.md) - Key feature descriptions
- [Tutorial Snippets](marketing/tutorial-snippets.md) - Quick tutorial content

### 🤖 GoMCP SDK
The MCP Memory Server uses the [GoMCP SDK](https://github.com/fredcamaral/gomcp-sdk) - a complete Go implementation of the Model Context Protocol that works with ANY MCP-compatible client.

- [GoMCP SDK Repository](https://github.com/fredcamaral/gomcp-sdk) - Open source MCP SDK for Go
- [SDK Documentation](https://pkg.go.dev/github.com/fredcamaral/gomcp-sdk) - API reference
- [SDK Examples](https://github.com/fredcamaral/gomcp-sdk/tree/main/examples) - Usage examples
- [Migration Guide](MCP_SDK_MIGRATION.md) - Migrating from embedded to standalone SDK

## 📂 Documentation Structure

```
docs/
├── README.md                    # This file - documentation index
├── ROADMAP.md                   # Project development roadmap
├── DEV-HOT-RELOAD.md           # Development environment setup
├── DEPLOYMENT.md               # Production deployment guide
├── MONITORING.md               # Monitoring and observability
├── ARCHITECTURE.md             # System architecture (planned)
├── TROUBLESHOOTING.md          # Troubleshooting guide (planned)
├── CONFIGURATION.md            # Configuration reference (planned)
├── website/                    # Website content
│   ├── index.md               # Homepage content
│   ├── getting-started.md     # Getting started guide
│   ├── api-reference.md       # API documentation
│   ├── examples.md            # Usage examples
│   └── faq.md                 # FAQ
├── ide/                        # IDE integration docs
│   ├── vscode-extension.md    # VS Code extension
│   └── intellij-plugin.md     # IntelliJ plugin
└── marketing/                  # Marketing materials
    ├── launch-announcement.md  # Launch messaging
    ├── feature-highlights.md   # Feature descriptions
    └── tutorial-snippets.md    # Tutorial content
```

## 🔗 External Resources

- [Model Context Protocol Specification](https://modelcontextprotocol.io) - Official MCP documentation
- [GitHub Repository](https://github.com/fredcamaral/mcp-memory) - Source code and issue tracking
- [Contributing Guidelines](../CONTRIBUTING.md) - How to contribute to the project

## 📝 Documentation Standards

When contributing to documentation:

1. Use clear, concise language
2. Include code examples where appropriate
3. Keep documentation up-to-date with code changes
4. Follow Markdown best practices
5. Test all examples and commands
6. Add cross-references to related documents

## 🆘 Need Help?

- Check the [FAQ](website/faq.md) for common questions
- Review the [Troubleshooting Guide](TROUBLESHOOTING.md) for known issues
- Open an [issue on GitHub](https://github.com/fredcamaral/mcp-memory/issues) for bugs or feature requests
- Contribute improvements via [pull requests](../CONTRIBUTING.md)

---

*Last updated: January 25, 2025*