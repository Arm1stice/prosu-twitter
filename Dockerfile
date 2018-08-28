FROM arm1stice/prosu-twitter

RUN export GOPATH=/go/

RUN go get github.com/golang/dep/cmd/dep

WORKDIR /go/src/github.com/wcalandro/prosu-go
COPY . /go/src/github.com/wcalandro/prosu-go

RUN dep ensure -vendor-only
RUN go build

ENTRYPOINT [ "./prosu-go" ]
EXPOSE 5000