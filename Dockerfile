FROM node:22.10.0-alpine3.20 AS web-builder

RUN npm install -g pnpm@9.9.0

WORKDIR /app/web

COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY web/ ./
RUN pnpm run build

FROM --platform=$BUILDPLATFORM golang:1.23-alpine3.20 AS app-builder

ARG VERSION=dev
ARG REVISION=dev
ARG BUILDTIME
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

RUN apk add --no-cache git build-base tzdata curl ca-certificates

ENV SERVICE=netronome
ENV CGO_ENABLED=0

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
COPY --from=web-builder /app/web/dist ./web/dist

RUN case "${TARGETOS}-${TARGETARCH}" in \
    "linux-amd64") \
        SPEEDTEST_ARCH="amd64"; \
        SPEEDTEST_CHECKSUM="78bf5cd10fc00006224efb82f5767b95ba414a7ca4dc315a7ed2b774ed80813c" ;; \
    "linux-arm64") \
        SPEEDTEST_ARCH="arm64"; \
        SPEEDTEST_CHECKSUM="e8d6fb054aee8bbfcad8c9f70fc89e983800f64221cfbb9b62ba1842bfcad6d4" ;; \
    "linux-386") \
        SPEEDTEST_ARCH="386"; \
        SPEEDTEST_CHECKSUM="392823ad7e984cb9aebce1922a5609cabdad3290c74a37f6dbf1384576adaa51" ;; \
    "linux-arm") \
        case "${TARGETVARIANT}" in \
            "v7") \
                SPEEDTEST_ARCH="armv7"; \
                SPEEDTEST_CHECKSUM="610aa869eb8db44599960fded5ce4e8833bbf332e3204ea998c2427bb47a271e" ;; \
            "v6") \
                SPEEDTEST_ARCH="armv6"; \
                SPEEDTEST_CHECKSUM="8772c020901e34ab28983cc1cc1b8b1c3244bb1db4453eaeafed6b00833f6291" ;; \
            "v5") \
                SPEEDTEST_ARCH="armv5"; \
                SPEEDTEST_CHECKSUM="d374b3bd9df8ab069c31b822e4f9e98409d26b1a77635dc6033173701854a338" ;; \
            *) \
                SPEEDTEST_ARCH="armv7"; \
                SPEEDTEST_CHECKSUM="610aa869eb8db44599960fded5ce4e8833bbf332e3204ea998c2427bb47a271e" ;; \
        esac ;; \
    *) echo "Unsupported platform: ${TARGETOS}-${TARGETARCH}" && exit 1 ;; \
    esac && \
    curl -fsSL --retry 3 --retry-delay 2 \
        -o /tmp/librespeed-cli.tar.gz \
        "https://github.com/librespeed/speedtest-cli/releases/download/v1.0.12/librespeed-cli_1.0.12_linux_${SPEEDTEST_ARCH}.tar.gz" && \
    echo "${SPEEDTEST_CHECKSUM}  /tmp/librespeed-cli.tar.gz" | sha256sum -c - && \
    tar -xzf /tmp/librespeed-cli.tar.gz -C /usr/local/bin/ && \
    chmod +x /usr/local/bin/librespeed-cli && \
    rm /tmp/librespeed-cli.tar.gz

RUN export GOOS=$TARGETOS; \
    export GOARCH=$TARGETARCH; \
    [[ "$GOARCH" == "amd64" ]] && export GOAMD64=$TARGETVARIANT; \
    [[ "$GOARCH" == "arm" ]] && [[ "$TARGETVARIANT" == "v6" ]] && export GOARM=6; \
    [[ "$GOARCH" == "arm" ]] && [[ "$TARGETVARIANT" == "v7" ]] && export GOARM=7; \
    echo "Building for: $GOARCH $GOOS $GOARM$GOAMD64"; \
    go build -ldflags "-s -w \
    -X 'main.version=${VERSION}' \
    -X 'main.commit=${REVISION}' \
    -X 'main.buildTime=${BUILDTIME}'" \
    -o /app/netronome cmd/netronome

# build runner
FROM alpine:latest


# Install dependencies
RUN apk add --no-cache sqlite iperf3 traceroute mtr tzdata

ENV HOME="/data" \
    XDG_CONFIG_HOME="/data" \
    XDG_DATA_HOME="/data"

WORKDIR /data

COPY --from=app-builder /app/netronome /usr/local/bin/netronome
COPY --from=app-builder /usr/local/bin/librespeed-cli /usr/local/bin/librespeed-cli

EXPOSE 7575

RUN addgroup -S netronome && \
    adduser -S netronome -G netronome && \
    mkdir -p /data && \
    chown -R netronome:netronome /data && \
    chmod 755 /data

USER netronome

ENTRYPOINT ["netronome"]
CMD ["serve"]