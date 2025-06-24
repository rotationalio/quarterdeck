# Dynamic Builds
ARG BUILDER_IMAGE=golang:1.23-bookworm
ARG FINAL_IMAGE=debian:bookworm-slim

# Build stage
FROM --platform=${BUILDPLATFORM} ${BUILDER_IMAGE} AS builder

# Build Args
ARG GIT_REVISION=""
ARG BUILD_DATE=""

# Platform Args
ARG TARGETOS
ARG TARGETARCH

# Ensure ca-certificates are up to date
RUN update-ca-certificates

# Use modules for dependencies
WORKDIR $GOPATH/src/go.rtnl.ai/quarterdeck

COPY go.mod .
COPY go.sum .

ENV CGO_ENABLED=1
ENV GO111MODULE=on
RUN go mod download
RUN go mod verify

# Copy package
COPY . .

# Build binary
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} CC=aarch64-linux-gnu-gcc go build \
    -ldflags="-X 'go.rtnl.ai/quarterdeck/pkg.GitVersion=${GIT_REVISION}' -X 'go.rtnl.ai/quarterdeck/pkg.BuildDate=${BUILD_DATE}'" \
    -o /go/bin/quarterdeck \
    ./cmd/quarterdeck

# Final Stage
FROM --platform=${BUILDPLATFORM} ${FINAL_IMAGE} AS final

LABEL maintainer="Rotational Labs <support@rotational.io>"
LABEL description="Quarterdeck authentication and authorization service"

# Ensure ca-certificates are up to date
RUN set -x && apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates sqlite3 && \
    rm -rf /var/lib/apt/lists/*

# Copy the binary to the production image from the builder stage
COPY --from=builder /go/bin/quarterdeck /usr/local/bin/quarterdeck

CMD [ "/usr/local/bin/quarterdeck", "serve" ]