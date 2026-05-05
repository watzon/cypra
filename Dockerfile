# syntax=docker/dockerfile:1.7

FROM oven/bun:1.3.11-alpine AS dashboard-build
WORKDIR /src
COPY package.json bun.lock ./
COPY dashboard/package.json dashboard/package.json
RUN bun install --frozen-lockfile
COPY dashboard dashboard
WORKDIR /src/dashboard
RUN bun run build

FROM golang:1.26.1-alpine AS go-build
WORKDIR /src
RUN apk add --no-cache ca-certificates git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=dashboard-build /src/dashboard/dist dashboard/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/cypra ./cmd/cypra

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=go-build /out/cypra /usr/local/bin/cypra
USER nonroot:nonroot
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 CMD ["/usr/local/bin/cypra", "version"]
ENTRYPOINT ["/usr/local/bin/cypra"]
CMD ["serve"]
