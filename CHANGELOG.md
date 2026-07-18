# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [1.0.0] - 2026-07-18

This is the initial release of the Greenbone Kubernetes deployment configuration tailored for airgapped environments.

### Added
- **GVM Helm Chart (`helm/greenbone/`)**:
  - Full-stack Kubernetes templates for Greenbone Security Assistant (`gsad` / Nginx), GVM Daemon (`gvmd`), OpenVAS scanner (`ospd-openvas`), and Redis.
  - Multi-container pod orchestration leveraging shared Unix domain sockets and volume structures.
  - CloudNativePG (CNPG) Cluster configuration mapped as a sub-chart dependency.
  - Dynamic resource rendering via `.Values.extraObjects` array for custom Traefik IngressRoutes and ServersTransports.
- **Go Data Fetcher API**:
  - In-pod API sidecar container exposing a secure `/upload` HTTP POST endpoint.
  - In-memory decompressor and streaming extractor for local GVM feed tarballs (`gvm-feed.tar.gz`).
  - Automatic path mapping to isolate and distribute files to respective GVM data folders (`plugins`, `scap-data`, `cert-data`, `data-objects`, `notus`).
  - Bearer Token authentication via `FETCH_AUTH_TOKEN` mapped to Kubernetes secrets.
- **GitHub Actions Workflow**:
  - Automated CI/CD workflow to build, package, sign (using cosign), and publish the `data-fetcher` container image to GitHub Container Registry (GHCR).
- **Documentation & Metadata**:
  - Detailed `README.md` explaining architecture and deployment procedures.
  - Comprehensive `.gitignore` to maintain clean commits.
  - MIT License specifying "as is" terms and disclaiming liability.
