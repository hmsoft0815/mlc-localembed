# Changelog

All notable changes to this project will be documented in this file.

## [1.0.1] - 2026-03-03

### Changed
- Refined GitHub Release workflow: Removed Intel Mac (`darwin-amd64`) build target to focus on modern architectures.
- Improved documentation: Renamed `problemspermodel.md` to `MODEL_GUIDE_DE.md` and updated it with best practices for v1.0.0.

### Fixed
- Fixed an issue where the `available` key was missing in `config.yaml`, causing models not to load.

## [1.0.0] - 2026-03-03

### Added
- **Ollama Parity**: Achieved mathematical parity with [Ollama](https://ollama.com/) embedding results.
- **Special Token Handling**: Automatically includes `[CLS]` and `[SEP]` tokens for BERT-based models (E5, BGE, MiniLM).
- **Automatic Truncation**: Inputs exceeding 512 tokens are now automatically truncated instead of returning an error, matching standard transformer behavior.
- New validation results documented in README: 1.000000 similarity for Nomic and 0.999999 for MiniLM.

### Changed
- Major version jump to reflect production readiness and verified accuracy.

## [0.3.9] - 2026-03-03

### Added
- Initial support for Nomic Embed text v1.5.
- Statistics API endpoint for monitoring.
- Concurrency protection for high-core systems.
- Preloader tool for easy model management.
