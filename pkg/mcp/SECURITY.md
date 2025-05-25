# Security Policy

## Supported Versions

We release patches for security vulnerabilities. Which versions are eligible for receiving such patches depends on the CVSS v3.0 Rating:

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We take the security of MCP-Go seriously. If you believe you have found a security vulnerability, please report it to us as described below.

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via email to security@example.com. You should receive a response within 48 hours. If for some reason you do not, please follow up via email to ensure we received your original message.

Please include the requested information listed below (as much as you can provide) to help us better understand the nature and scope of the possible issue:

- Type of issue (e.g. buffer overflow, SQL injection, cross-site scripting, etc.)
- Full paths of source file(s) related to the manifestation of the issue
- The location of the affected source code (tag/branch/commit or direct URL)
- Any special configuration required to reproduce the issue
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit the issue

This information will help us triage your report more quickly.

## Preferred Languages

We prefer all communications to be in English.

## Policy

We follow the principle of Coordinated Vulnerability Disclosure.

## Security Measures

MCP-Go implements several security measures:

### Input Validation
- All inputs are validated before processing
- JSON schema validation for tool inputs
- Path traversal prevention for file operations
- SQL injection prevention (where applicable)

### Authentication & Authorization
- JWT token validation
- API key authentication
- Rate limiting to prevent abuse
- Context-based access control

### Secure Defaults
- TLS enabled by default for HTTP transport
- Secure random number generation
- Constant-time comparison for secrets
- No default credentials

### Dependencies
- Regular dependency updates
- Vulnerability scanning in CI/CD
- License compliance checking

## Security Checklist for Contributors

When contributing code, please ensure:

- [ ] No hardcoded secrets or credentials
- [ ] Input validation for all user inputs
- [ ] Error messages don't leak sensitive information
- [ ] Proper authentication/authorization checks
- [ ] No use of deprecated or insecure functions
- [ ] Dependencies are up to date
- [ ] Security tests are included where appropriate

## Acknowledgments

We appreciate the security research community's efforts in helping keep MCP-Go secure. Responsible disclosure of vulnerabilities helps us ensure the security and privacy of our users.

## Contact

- Security Email: security@example.com
- PGP Key: [Link to PGP key]