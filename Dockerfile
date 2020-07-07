FROM arm1stice/prosu-twitter:alpine

RUN export GOPATH=/go/

WORKDIR /go/src/github.com/Arm1stice/prosu-twitter
COPY . /go/src/github.com/Arm1stice/prosu-twitter
COPY CHECKS /app/CHECKS

RUN go mod vendor
RUN go build

ENTRYPOINT [ "./prosu-twitter" ]
EXPOSE 5000