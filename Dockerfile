FROM golang:1.13-alpine

LABEL maintainer="Sam Silverberg  <sam.silverberg@gmail.com>"

ENV GOPATH /go
ENV CGO_ENABLED 0
ENV GO111MODULE on
COPY ./ /go/sdfs-client-go

RUN  \
     apk add --no-cache git && \
     cd /go/sdfs-client-go && \
     mkdir -p /go/sdfs-client-go/build && \
     go build -o /go/sdfs-client-go/build/sdfscli app/*
