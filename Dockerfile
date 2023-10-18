# syntax=docker/dockerfile:1.3

FROM golang:1.20.5-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /build

FROM alpine AS release 

RUN     apk add doas; \
        adduser golang; \
        echo 'golang:123' | chpasswd; \
        echo 'permit golang as root' > /etc/doas.d/doas.conf

USER golang

WORKDIR /app

COPY --from=builder /build ./
COPY  config.yml ./ 

EXPOSE 9100

CMD ["./build"]