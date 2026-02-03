FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -trimpath \
    -o laps .

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata && \
    update-ca-certificates && \
    adduser -D -u 1000 appuser

WORKDIR /app

COPY --from=builder /app/laps .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/docs ./docs

RUN chown -R appuser:appuser /app

USER appuser

ENV GIN_MODE=release

EXPOSE 8080

CMD ["./laps"] 