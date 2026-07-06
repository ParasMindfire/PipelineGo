# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder
WORKDIR /app

RUN apk --no-cache add git

# Download dependencies first (cached layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy entire repo (go.mod is at root and references packages/ and apps/)
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /server ./apps/server

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Pipeline writes job output here; mount a volume over this in compose
RUN mkdir -p data/output

COPY --from=builder /server .

EXPOSE 8080
CMD ["./server"]
