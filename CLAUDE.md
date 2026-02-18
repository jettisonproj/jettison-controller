# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Jettison Controller is a Kubernetes operator that orchestrates CI/CD workflows by managing resources across the Argo ecosystem (Workflows, CD, Events). It bridges GitHub webhooks to automated deployment pipelines via a `Flow` custom resource.

## Common Commands

### Build & Test
```bash
make build          # Build manager binary to bin/manager
make test           # Run unit tests with coverage (requires envtest assets)
make test-e2e       # Run end-to-end tests (requires a Kubernetes cluster)
make lint           # Run golangci-lint
make lint-fix       # Run golangci-lint with auto-fix
```

### Code Generation (run after modifying API types)
```bash
make manifests      # Regenerate CRDs, RBAC manifests
make generate       # Regenerate DeepCopy methods
```

### Local Development
```bash
make run            # Run controller locally (see Makefile for required flags)
make install        # Install CRDs into cluster
make deploy         # Deploy controller to cluster
```

### Running a Single Test
```bash
go test ./internal/controller/... -run TestFunctionName -v
go test ./api/v1alpha1/... -run TestFunctionName -v
```

## Architecture

### Flow CRD → Argo Resources
The core pattern is: a user creates a `Flow` resource, and the FlowReconciler generates and reconciles Argo ecosystem resources from it.

```
Flow CR
  ├── triggers (GitHub PR/Push) → Argo Events EventSource + Sensor
  └── steps (DockerBuildTest, DockerBuildTestPublish, ArgoCD) → Argo Workflows + Argo CD Applications/AppProjects
```

### Key Components

**`cmd/main.go`** — Entry point. Initializes the controller-runtime manager, registers FlowReconciler and webhooks, and starts the web server.

**`api/v1alpha1/`** — CRD API types. `Flow` spec stores triggers and steps as raw JSON, parsed by `flow_parser.go` into typed objects. `flow_validator.go` and `flow_defaulter.go` handle admission webhooks.

**`internal/controller/flow_controller.go`** — Main reconciliation loop. Maintains in-memory sync state (synced namespaces, projects, repos) to avoid unnecessary API calls. Authenticates with GitHub and ArgoCD.

**Builder packages** (generate Argo resources from Flow spec):
- `internal/controller/appbuilder/` — Flow steps → Argo CD Applications & AppProjects
- `internal/controller/sensorbuilder/` — Flow triggers → Argo Events Sensors
- `internal/controller/eventsourcebuilder/` — Flow triggers → GitHub webhook EventSources

**`internal/webserver/`** — HTTP API server (port `:2846`) serving Flow data. Includes WebSocket support for real-time updates and MySQL integration for workflow history.

**`internal/workflowtemplates/`** — Syncs ClusterWorkflowTemplates to the cluster.

**`internal/controller/ghsettings/`** — GitHub App authentication via private key.

### Runtime Flags
Key flags for local development:
- `-workflow-mysql-address` — MySQL DSN (default: `/argo`)
- `-github-key` — Path to GitHub App private key PEM file
- `-argocd-key` — Path to ArgoCD password file
- `-server-bind-address` — HTTP server (default: `:2846`)

### Dependencies
- **Argo**: `argo-cd/v3`, `argo-workflows/v3`, `argo-events`, `argo-rollouts`
- **Kubernetes**: `controller-runtime`, `client-go`
- **GitHub**: `go-github/v74`, `ghinstallation/v2` (GitHub App auth)
- **Storage**: MySQL via `go-sql-driver/mysql`

### CRD Details
- API group: `workflows.jettisonproj.io`
- CRD domain: `jettisonproj.io`
- Leader election ID: `3d4b3c11.jettisonproj.io`
