# MCP-Go Documentation

Welcome to the MCP-Go library documentation! This directory contains comprehensive documentation for the production-ready Go implementation of the Model Context Protocol.

## 📚 Documentation Index

### 🚀 Getting Started
- [Library Overview](../README.md) - Main README with features and quick start
- [Tutorial](guides/TUTORIAL.md) - Step-by-step guide to building your first MCP server
- [Examples](../examples/README.md) - Working examples of MCP servers

### 📖 Developer Guides
- [API Design Principles](reference/API_DESIGN.md) - Core design philosophy and patterns
- [Advanced Usage](guides/ADVANCED.md) - Advanced features and patterns
- [Performance Guide](guides/PERFORMANCE.md) - Optimization tips and benchmarks
- [Breaking Changes](reference/BREAKING_CHANGES.md) - Migration guides for API changes

### 🔌 Integration Guides
- [Claude Integration](integration/INTEGRATION_CLAUDE.md) - Integrating with Claude Desktop
- [LLM Integration](integration/INTEGRATION_LLM.md) - Integrating with other LLMs
- [Client Integration](integration/INTEGRATION_CLIENT.md) - Building MCP clients

### 🛠️ Operations
- [Deployment Guide](operations/DEPLOYMENT.md) - Production deployment best practices
- [Monitoring & Observability](operations/MONITORING.md) - Metrics, tracing, and logging
- [Security Guide](operations/SECURITY.md) - Security best practices
- [Security Audit](operations/SECURITY_AUDIT.md) - Security checklist

### 🏛️ Project Governance
- [Contributing Guidelines](../CONTRIBUTING.md) - How to contribute to MCP-Go
- [Community Guidelines](governance/COMMUNITY.md) - Community participation
- [Governance Model](governance/GOVERNANCE.md) - Project governance structure
- [Beta Testing Program](governance/BETA_TESTING.md) - Early access program
- [Launch Checklist](governance/LAUNCH_CHECKLIST.md) - Release preparation

### 📋 Project Management
- [Roadmap](../ROADMAP.md) - Development roadmap and future plans
- [Changelog](../CHANGELOG.md) - Version history and release notes
- [License](../LICENSE) - MIT License

## 📂 Documentation Structure

```
pkg/mcp/docs/
├── README.md                      # This file - documentation index
├── guides/                        # Developer guides and tutorials
│   ├── TUTORIAL.md               # Step-by-step tutorial
│   ├── ADVANCED.md               # Advanced usage patterns
│   └── PERFORMANCE.md            # Performance optimization
├── reference/                     # API and technical reference
│   ├── API_DESIGN.md             # API design principles
│   └── BREAKING_CHANGES.md       # Breaking change documentation
├── integration/                   # Integration guides
│   ├── INTEGRATION_CLAUDE.md    # Claude Desktop integration
│   ├── INTEGRATION_LLM.md       # LLM integration guide
│   └── INTEGRATION_CLIENT.md    # Client development guide
├── operations/                    # Operational documentation
│   ├── DEPLOYMENT.md             # Deployment guide
│   ├── MONITORING.md             # Monitoring setup
│   ├── SECURITY.md               # Security practices
│   └── SECURITY_AUDIT.md         # Security checklist
└── governance/                    # Project governance
    ├── COMMUNITY.md              # Community guidelines
    ├── GOVERNANCE.md             # Governance model
    ├── BETA_TESTING.md           # Beta program
    └── LAUNCH_CHECKLIST.md       # Launch preparation
```

## 🔧 Key Resources

### Code Organization
- `protocol/` - Core MCP protocol types and interfaces
- `server/` - Server implementation
- `transport/` - Transport layer (stdio, HTTP, WebSocket)
- `middleware/` - Middleware components
- `examples/` - Example implementations

### Tools
- `tools/mcp-validator` - MCP protocol validator
- `tools/mcp-benchmark` - Performance benchmarking tool

### Deployment Assets
- `Dockerfile` - Production Docker image
- `docker-compose.yml` - Docker Compose configuration
- `kubernetes/` - Kubernetes deployment manifests
- `helm/` - Helm charts

## 📊 Library Status

- **Current Version**: See [version.go](../version.go)
- **API Stability**: Production-ready (see [api_stability.go](../api_stability.go))
- **Protocol Compliance**: MCP 2024-11-05
- **Test Coverage**: 90%+

## 🆘 Getting Help

1. Check the [Tutorial](guides/TUTORIAL.md) for getting started
2. Review [Examples](../examples/) for working code
3. Read the [API Design](reference/API_DESIGN.md) for patterns
4. Open an issue on GitHub for bugs or questions

## 🤝 Contributing

We welcome contributions! Please see:
- [Contributing Guidelines](../CONTRIBUTING.md)
- [Community Guidelines](governance/COMMUNITY.md)
- [Governance Model](governance/GOVERNANCE.md)

---

*Last updated: January 25, 2025*