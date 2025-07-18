# Build the manager binary
FROM registry.access.redhat.com/ubi9/go-toolset:1.24.4-1752083840 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
USER 0
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build ./cmd/mbop/mbop.go

FROM registry.access.redhat.com/ubi9-minimal
WORKDIR /
COPY --from=builder /workspace/mbop .
USER 65532:65532

ENTRYPOINT ["/mbop"]
