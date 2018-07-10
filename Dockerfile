FROM golang:alpine AS build
RUN apk add --no-cache git
RUN go get github.com/golang/dep/cmd/dep
RUN go get -u github.com/gobuffalo/packr/...

COPY Gopkg.lock Gopkg.toml /go/src/github.com/wcalandro/prosu-go/
WORKDIR /go/src/github.com/wcalandro/prosu-go/
RUN dep ensure -vendor-only

COPY . /go/src/github.com/wcalandro/prosu-go

RUN go build -o ./link-shortener

ENTRYPOINT "./link-shortener" 
EXPOSE 8080