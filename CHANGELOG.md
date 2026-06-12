# Changelog

All notable changes to this project will be documented in this file.

## [1.5.1] - 2026-06-12

### Added
- **Native systemd installer** for non-RPM Linux (`scripts/install-linux-systemd.sh`):
  installs binaries, the ONNX lib, config, a `localembed` service user, the
  systemd unit, seeds models, and starts the service. Verified on Ubuntu.

### Fixed
- **RPM packaging**: the `localembed` RPM now builds, installs, and serves
  embeddings **offline out of the box** (verified on Fedora 41 with no network).
  Fixed version drift, binary/man-page naming, the `/var/lib` state dir,
  host-independent systemd scriptlets, and self-provided `user`/`group(localembed)`
  to satisfy rpm >= 4.19 auto-deps.
- The RPM now **bundles the default model** (`multilingual-e5-small`) so the
  service starts immediately instead of exiting on an empty models dir.
- `config.yaml.template` default model now resolves correctly.
- Build artifacts (`build/`, `libtokenizers.a`) are now gitignored.

## [1.4.0] - 2026-03-13

### Added
- **GPU Acceleration**: Optional support for ONNX Runtime Execution Providers.
- Platform auto-detection: CoreML (macOS), CUDA (Linux), DirectML (Windows).
- New configuration options: `use_gpu` and `execution_provider` in `config.yaml`.
- Command-line flags `-gpu` and `-gpu-ep` for server and CLI.
- Environment variables `MLC_USE_GPU` and `MLC_GPU_EP`.
- Graceful fallback to CPU if GPU initialization fails.

## [1.0.2] - 2026-03-03

### Changed
- Bumped version to ensure clean GitHub Release build and avoid tag caching issues (HTTP 500 on fetch).
- Updated README with complete `config.yaml` example including storage and migration settings.

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
