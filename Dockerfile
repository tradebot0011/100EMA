FROM golang:1.22-alpine AS builder
WORKDIR /app

# Copy module definition
COPY go.mod ./

# Clear cache and fetch fresh checksums inside the container
RUN go mod tidy

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o trading-bot main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/trading-bot .
CMD ["./trading-bot"]
