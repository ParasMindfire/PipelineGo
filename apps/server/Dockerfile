FROM golang:1.26.4 AS builder
WORKDIR /app

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOTOOLCHAIN=local
ENV GONOSUMDB=*
ENV GOFLAGS=-mod=mod

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /server ./apps/server

FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
RUN mkdir -p data/output
COPY --from=builder /server .
RUN chown -R nobody:nobody /app
USER nobody
EXPOSE 8080
CMD ["./server"]
