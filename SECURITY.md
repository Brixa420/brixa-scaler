# Security Policy

## Reporting Vulnerabilities

If you discover a security vulnerability, please report it responsibly. Do not open public issues for security vulnerabilities.

**Report via:** Private message to maintainers or security contact

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |

## Known Limitations

- ZK circuits not yet independently audited
- No formal verification of critical code paths
- Demo mode enabled by default for safety

## Threat Model

### In Scope
- Transaction batching and Merkle tree construction
- ZK proof generation and verification
- API endpoints for batch submission

### Out of Scope
- Blockchain node security
- KMS/hardware wallet integration (not yet implemented)
- HTTPS termination (should be handled by reverse proxy)

## Security Hardening for Production

See README.md "Detailed Security Requirements" section for production hardening guidelines.

## Changelog

### 1.0.0
- Initial release
- Demo mode enabled by default
- No production security audit completed
