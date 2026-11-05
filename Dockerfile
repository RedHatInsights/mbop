# Build the manager binary
# Red Hat Hardened Images: https://access.redhat.com/documentation/en-us/red_hat_hardened_images
FROM registry.access.redhat.com/hi/go:1.26.10-builder@sha256:4a291ab31e4a8678c3951a5567e99c4a4b46b9843a96e9d098dc895ac3e31334 AS builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY --chown=65532:0 go.mod go.sum ./
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
ENV GOTOOLCHAIN=auto
RUN go mod download

COPY --chown=65532:0 . .
RUN CGO_ENABLED=0 go build ./cmd/mbop/mbop.go

FROM registry.access.redhat.com/hi/static:1792647314@sha256:7e14ec7af494b367c780b040122a685071f73e1f766c28f4390ab9ed2fd4e333

WORKDIR /
COPY --from=builder --chown=65532:0 /workspace/mbop .
USER 65532:65532

ENTRYPOINT ["/mbop"]
