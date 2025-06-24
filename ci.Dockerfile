FROM --platform=$BUILDPLATFORM golang:1.23-alpine3.20 AS app-builder

ARG VERSION=dev
ARG REVISION=dev
ARG BUILDTIME
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

RUN apk add --no-cache git tzdata

ENV SERVICE=netronome
ENV CGO_ENABLED=0

WORKDIR /src

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

COPY . ./

# Build with platform-specific settings
RUN --network=none --mount=target=. \
    export GOOS=$TARGETOS
export GOARCH=$TARGETARCH
[[ "$GOARCH" == "amd64" ]] && export GOAMD64=$TARGETVARIANT
[[ "$GOARCH" == "arm" ]] && [[ "$TARGETVARIANT" == "v6" ]] && export GOARM=6
[[ "$GOARCH" == "arm" ]] && [[ "$TARGETVARIANT" == "v7" ]] && export GOARM=7
echo "Building for: $GOARCH $GOOS $GOARM$GOAMD64"
go build -ldflags "-s -w \
    -X netronome/internal/buildinfo.Version=${VERSION} \
    -X netronome/internal/buildinfo.Commit=${REVISION} \
    -X netronome/internal/buildinfo.Date=${BUILDTIME}" \
    -o /app/netronome cmd/netronome/main.go

FROM alpine:latest

LABEL org.opencontainers.image.source="https://github.com/autobrr/netronome"
LABEL org.opencontainers.image.licenses="GPL-2.0-or-later"
LABEL org.opencontainers.image.base.name="alpine:latest"

RUN apk add --no-cache sqlite iperf3

ENV HOME="/data" \
    XDG_CONFIG_HOME="/data" \
    XDG_DATA_HOME="/data"

WORKDIR /data

COPY --from=app-builder /app/netronome /usr/local/bin/netronome

EXPOSE 7575

RUN addgroup -S netronome &&
    adduser -S netronome -G netronome &&
    mkdir -p /data &&
    chown -R netronome:netronome /data &&
    chmod 755 /data

USER netronome

ENTRYPOINT ["netronome"]
CMD ["serve"]
