FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -trimpath -o /app/virtsync ./cmd/sync/main.go

FROM alpine:3.21.3

RUN apk --no-cache update && apk --no-cache add stress-ng

USER 1001
WORKDIR /app

COPY --from=builder /app/virtsync .

CMD ["/app/virtsync"]
