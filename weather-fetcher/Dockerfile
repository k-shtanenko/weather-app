FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o weather-fetcher ./cmd/weather-fetcher

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/weather-fetcher .
COPY --from=builder /app/config.yaml .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

RUN apk add --no-cache tzdata
ENV TZ=Europe/Moscow

CMD ["./weather-fetcher"]