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


LABEL org.opencontainers.image.source="https://github.com/autobrr/netronome"
LABEL org.opencontainers.image.licenses="GPL-2.0-or-later"
LABEL org.opencontainers.image.base.name="alpine:latest"


# Install dependencies
RUN apk add --no-cache sqlite iperf3

ENV HOME="/data" \
    XDG_CONFIG_HOME="/data" \
    XDG_DATA_HOME="/data"

WORKDIR /data

COPY --from=app-builder /app/netronome /usr/local/bin/netronome

EXPOSE 7575

RUN addgroup -S netronome && \
    adduser -S netronome -G netronome && \
    mkdir -p /data && \
    chown -R netronome:netronome /data && \
    chmod 755 /data

USER netronome

ENTRYPOINT ["netronome"]
CMD ["serve"]
