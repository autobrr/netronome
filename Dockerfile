FROM node:22.10.0-alpine3.20 AS web-builder

# Install specific pnpm version instead of using corepack
RUN npm install -g pnpm@9.9.0

WORKDIR /app/web

COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY web/ ./
RUN pnpm run build

FROM golang:1.23-alpine3.20 AS app-builder

ARG VERSION=dev
ARG REVISION=dev
ARG BUILDTIME

RUN apk add --no-cache git build-base tzdata

ENV SERVICE=netronome
ENV CGO_ENABLED=0

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
COPY --from=web-builder /app/web/dist ./web/dist

RUN go build -ldflags "-s -w \
    -X netronome/internal/buildinfo.Version=${VERSION} \
    -X netronome/internal/buildinfo.Commit=${REVISION} \
    -X netronome/internal/buildinfo.Date=${BUILDTIME}" \
    -o /app/netronome cmd/netronome/main.go

# build runner
FROM alpine:latest

# Install dependencies
RUN apk add --no-cache sqlite iperf3 curl &&
    LIBRESPEED_VERSION=1.0.12 &&
    curl -L -o /tmp/librespeed-cli.tar.gz https://github.com/librespeed/speedtest-cli/releases/download/v${LIBRESPEED_VERSION}/librespeed-cli_${LIBRESPEED_VERSION}_linux_amd64.tar.gz &&
    tar -C /usr/local/bin -xzf /tmp/librespeed-cli.tar.gz librespeed-cli &&
    rm /tmp/librespeed-cli.tar.gz &&
    chmod +x /usr/local/bin/librespeed-cli &&
    apk del curl

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
