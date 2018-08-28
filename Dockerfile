FROM arm1stice/prosu-twitter

RUN go build

ENTRYPOINT [ "./prosu-go" ]