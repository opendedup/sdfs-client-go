FROM golang:alpine3.15

LABEL maintainer="Sam Silverberg  <sam.silverberg@gmail.com>"

ENV GOPATH /go
ENV CGO_ENABLED 0
ENV GO111MODULE on
COPY ./ /go/sdfs-client-go

RUN  \
     apk add --no-cache git build-base && \
     cd /go/sdfs-client-go && \
     mkdir -p /go/sdfs-client-go/build
WORKDIR /go/sdfs-client-go/
RUN make clean && make build
