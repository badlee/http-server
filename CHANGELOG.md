# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.2] - 2026-04-20

### Added
- Multi-platform release script (`scripts/release.sh`) with GitHub Release automation.
- Windows Named Pipe support and cross-platform compatibility.

### Fixed
- Windows compilation errors related to Unix-specific socket syscalls (FD passing).
- Abstracted File Descriptor passing logic for consistent behavior across Linux, Mac, and Windows.

## [0.0.1] - 2026-04-19

### Added
- Initial release of **beba**.
- Multi-protocol support: HTTP, HTTPS, TCP, UDP.
- Real-time engine with SSE, WebSocket, and MQTT integration.
- Advanced VHost management with `.vhost.bind` configurations.
- Embedded JavaScript engine for dynamic routing and event handling.
- Integrated Coraza WAF for security.
- ClickHouse, Postgres, MySQL, and SQLite database connectors.
