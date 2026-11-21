# syntax=docker/dockerfile:1

# Shared base: toolchain, module download, and common sources.
# Runs on the build host's native platform; binaries are cross-compiled via GOOS/GOARCH.
FROM --platform=$BUILDPLATFORM golang:1.27.1 AS base
WORKDIR /workspace
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY api/ api/
COPY internal/ internal/

# Build the manager binary
FROM base AS build
ARG TARGETOS TARGETARCH
COPY cmd/ cmd/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/manager cmd/main.go

# Build the integration test binary
FROM base AS build-e2e
ARG TARGETOS TARGETARCH
COPY test/ test/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go test -c -trimpath -o /out/e2e.test ./test/e2e

# Lint and unit test (runs natively on the build host)
FROM base AS test-results
COPY cmd/ cmd/
COPY test/ test/
COPY testdata/ testdata/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    FORMAT_RESULT="$(gofmt -d -e -s api cmd internal test/e2e)" && \
    echo "${FORMAT_RESULT}" && \
    test -z "${FORMAT_RESULT}" && \
    go vet ./... && \
    go tool staticcheck ./... && \
    go fix -diff ./... && \
    go test $(go list ./... | grep -v /e2e) -coverprofile cover.out

# Integration Test
# Use distroless as minimal base image to package the integration test binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot AS integration-test
WORKDIR /test/e2e
COPY --link --from=build-e2e /out/e2e.test /test/e2e/e2e.test
COPY --link testdata/ /testdata/
USER 65532:65532

ENTRYPOINT ["/test/e2e/e2e.test"]

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --link --from=build /out/manager /manager
USER 65532:65532

ENTRYPOINT ["/manager"]
