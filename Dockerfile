FROM golang:alpine

RUN apk update && \
    apk add --no-cache \
    --repository http://dl-3.alpinelinux.org/alpine/edge/testing \
    pkgconfig fftw-dev build-base vips-dev imagemagick && \
    rm -rf /var/cache/apk/*

COPY vendor/gopkg.in/h2non/bimg.v1 src/gopkg.in/h2non/bimg.v1

COPY vendor/github.com src/github.com

COPY main.go /go/main.go

RUN GOOS=linux GOARCH=amd64 go build

CMD ./go