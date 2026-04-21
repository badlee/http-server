# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.1] - 2026-04-20

### Added
- Initial release of **beba**.
- Multi-protocol support: HTTP, HTTPS, TCP, UDP.
- Automated HTTPS certificate fallback: Let's Encrypt (ACME) with self-signed auto-generation.
- New configuration fields: `Domain` and `Email` for global ACME support.
- Improved console output with detailed HTTPS status (Cert/ACME/Self-signed) and URL list.
- Refined ACME manager to handle optional email addresses.
- Automatic initialization of default tables for `users`, `roles`, and `permissions` in the default database (`.data/beba.db`).
- Improved relationship support in the internal CRUD engine, including `many2many` join table creation.
- Real-time engine with SSE, WebSocket, and MQTT integration.
- Advanced VHost management with `.vhost.bind` configurations.
- Embedded JavaScript engine for dynamic routing and event handling.
- Integrated Coraza WAF for security.
- ClickHouse, Postgres, MySQL, and SQLite database connectors.
- Multi-platform release script (`scripts/release.sh`) with GitHub Release automation.
- Windows Named Pipe support and cross-platform compatibility.
- **Default CRUD API and Admin UI**: Automatically mounted at `/api` and `/api/_admin` on startup for the default database, enabling immediate user and role management.
- **Improved CRUD route matching**: Support for base paths without trailing slashes (e.g., `/api/namespaces`).

### Fixed
- Windows compilation errors related to Unix-specific socket syscalls (FD passing).
- Abstracted File Descriptor passing logic for consistent behavior across Linux, Mac, and Windows.
- Fixed release script to upload only release assets.
- Fixed default values for `cert` and `key` to be empty.
