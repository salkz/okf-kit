# Image of the okf tool. The release workflow builds it for each tag:
#   docker run --rm -u "$(id -u):$(id -g)" -v "$PWD":/repo ghcr.io/salkz/okf-kit init
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG VERSION=dev
ARG TARGETOS=linux
ARG TARGETARCH
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /okf ./cmd/okf

# The binary is static. It needs certificates only for `okf status`, which
# asks GitHub for the newest release.
FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /okf /okf
WORKDIR /repo
EXPOSE 8765
ENTRYPOINT ["/okf"]
