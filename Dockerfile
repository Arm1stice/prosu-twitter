FROM golang:alpine AS build
RUN apk add --no-cache git
RUN go get github.com/golang/dep/cmd/dep

COPY . /go/src/github.com/wcalandro/prosu-go/
WORKDIR /go/src/github.com/wcalandro/prosu-go/
RUN dep ensure -vendor-only

RUN go build

ENTRYPOINT "./prosu-go" 
EXPOSE 5000