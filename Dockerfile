FROM golang:1.17 AS builder

ENV GO111MODULE=on

WORKDIR /go/src/github.com/pando85/crow

ADD . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o crow .


FROM alpine:latest

RUN \
    apk add --no-cache ca-certificates ;\
    mkdir -p /opt/crow/static

WORKDIR /opt/crow

COPY --from=builder /go/src/github.com/pando85/crow/crow .
COPY static /opt/crow/static

CMD [ "./crow" ]
