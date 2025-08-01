# Stage 1: Build
FROM golang:1.25-rc-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /main .


# Stage 2: Run
FROM alpine:latest

WORKDIR /

COPY --from=builder /main /main

EXPOSE 8080

CMD ["/main"]