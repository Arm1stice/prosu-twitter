FROM arm1stice/prosu-twitter

RUN export GOPATH=/go/

RUN go get github.com/golang/dep/cmd/dep

WORKDIR /go/src/github.com/wcalandro/prosu-twitter
COPY . /go/src/github.com/wcalandro/prosu-twitter
COPY CHECKS /app/CHECKS

RUN dep ensure -vendor-only
RUN go build

ENTRYPOINT [ "./prosu-twitter" ]
EXPOSE 5000