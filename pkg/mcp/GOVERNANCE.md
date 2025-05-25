# MCP-Go Governance

This document outlines the governance structure for the MCP-Go project.

## Project Roles

### Maintainers

Maintainers are responsible for:
- Reviewing and merging pull requests
- Making architectural decisions
- Managing releases
- Enforcing code of conduct
- Growing the contributor community

Current maintainers:
- [Your Name] (@yourusername) - Project Lead
- [Maintainer 2] (@username2)
- [Maintainer 3] (@username3)

### Committers

Committers have write access and can:
- Merge approved pull requests
- Triage issues
- Participate in design decisions

### Contributors

Anyone who contributes to the project through:
- Code contributions
- Documentation improvements
- Bug reports
- Feature requests
- Community support

## Decision Making

### Consensus Model

Decisions are made through consensus among maintainers. We strive for unanimous agreement but will proceed with majority approval when necessary.

### Types of Decisions

1. **Minor Changes**: Bug fixes, documentation updates
   - Single maintainer approval required
   - 24-hour review period

2. **Major Changes**: New features, API changes
   - Two maintainer approvals required
   - 72-hour review period
   - Documented in ADR (Architecture Decision Record)

3. **Breaking Changes**: Incompatible API changes
   - All maintainer approval required
   - 1-week review period
   - Migration guide required

### Architecture Decision Records (ADRs)

Significant architectural decisions are documented in ADRs stored in `docs/adr/`:

```
docs/adr/
├── 0001-use-json-rpc.md
├── 0002-plugin-architecture.md
└── template.md
```

## Becoming a Maintainer

Contributors can become maintainers through:

1. **Sustained Contributions**: 6+ months of active contribution
2. **Technical Expertise**: Deep understanding of the codebase
3. **Community Involvement**: Helping other contributors
4. **Nomination**: By existing maintainer
5. **Vote**: Unanimous approval from current maintainers

## Code of Conduct Enforcement

See CODE_OF_CONDUCT.md for our code of conduct. Violations are handled by:

1. **First Violation**: Warning from maintainers
2. **Second Violation**: Temporary ban (1 week to 1 month)
3. **Third Violation**: Permanent ban

Appeals can be made to: conduct@example.com

## Release Process

### Release Schedule

- **Major Releases**: Annually (x.0.0)
- **Minor Releases**: Quarterly (0.x.0)
- **Patch Releases**: As needed (0.0.x)

### Release Approval

- Patch releases: 1 maintainer
- Minor releases: 2 maintainers
- Major releases: All maintainers

### Release Steps

1. Create release branch
2. Update CHANGELOG.md
3. Run full test suite
4. Create release PR
5. Get required approvals
6. Merge and tag
7. Publish release notes
8. Update documentation

## Communication Channels

- **GitHub Issues**: Bug reports, feature requests
- **GitHub Discussions**: General discussions
- **Discord**: Real-time chat (#mcp-go channel)
- **Mailing List**: mcp-go@example.com
- **Security Issues**: security@example.com

## Meetings

- **Monthly Maintainer Meeting**: First Tuesday of each month
- **Quarterly Community Call**: Open to all contributors
- **Annual Roadmap Planning**: January

Meeting notes are published in `docs/meetings/`.

## Roadmap Planning

The project roadmap is planned:

1. **Annual Planning**: Major version goals
2. **Quarterly Review**: Adjust priorities
3. **Community Input**: Through issues and discussions
4. **Roadmap Location**: `ROADMAP.md`

## Conflict Resolution

Conflicts are resolved through:

1. **Discussion**: In relevant issue/PR
2. **Mediation**: By uninvolved maintainer
3. **Vote**: If consensus cannot be reached
4. **Escalation**: To project lead as last resort

## Amendments

This governance document can be amended through:

1. Proposal via pull request
2. 2-week discussion period
3. Unanimous maintainer approval
4. 30-day notice before taking effect

## Acknowledgments

This governance model is inspired by:
- Apache Software Foundation
- Cloud Native Computing Foundation
- Node.js Foundation

Last Updated: [Date]