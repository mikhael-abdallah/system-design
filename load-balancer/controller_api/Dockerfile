# Stage 1: Build
FROM golang:1.25-rc-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /controller-api

# Stage 2: Run
FROM alpine:latest
WORKDIR /
COPY --from=builder /controller-api /controller-api
EXPOSE 8000
CMD ["/controller-api"]