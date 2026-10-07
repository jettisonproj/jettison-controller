# Build the manager binary
FROM golang:1.27.1 AS build
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

# Download additional go linter
RUN go install honnef.co/go/tools/cmd/staticcheck@latest

# Copy the go source
COPY go.mod go.mod
COPY go.sum go.sum
COPY cmd/main.go cmd/main.go
COPY api/ api/
COPY internal/ internal/
COPY test/ test/
COPY testdata/ testdata/

# Build Main Executable and Integration Test Executable
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager cmd/main.go && \
  CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go test -c ./test/e2e

# Unit Test
FROM build as test-results
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  FORMAT_RESULT="$(gofmt -d -e -s api cmd internal test/e2e)" && \
  echo "${FORMAT_RESULT}" && \
  test -z "${FORMAT_RESULT}" && \
  go vet ./... && \
  staticcheck ./... && \
  go fix -diff ./... && \
  go test $(go list ./... | grep -v /e2e) -coverprofile cover.out

# Integration Test
# Use distroless as minimal base image to package the integration test binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot as integration-test
WORKDIR /test/e2e
COPY --from=build /workspace/e2e.test /test/e2e/e2e.test
COPY testdata/ /testdata/
USER 65532:65532

ENTRYPOINT ["/test/e2e/e2e.test"]

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
