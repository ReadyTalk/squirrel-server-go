FROM golang

ADD . /go/src/github.com/readytalk/squirrel-server-go

RUN go install github.com/readytalk/squirrel-server-go

ENTRYPOINT /go/bin/squirrel-server-go

EXPOSE 3000
