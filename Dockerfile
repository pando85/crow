FROM golang:1.17 AS builder

ENV GO111MODULE=on

WORKDIR /go/src/github.com/pando85/crow

ADD . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o crow .


FROM alpine:latest

ENV \
    VAULT_ADDR \
    VAULT_TOKEN \
    SUPERSECRETMESSAGE_HTTP_BINDING_ADDRESS \
    SUPERSECRETMESSAGE_HTTPS_BINDING_ADDRESS \
    SUPERSECRETMESSAGE_HTTPS_REDIRECT_ENABLED \
    SUPERSECRETMESSAGE_TLS_AUTO_DOMAIN \
    SUPERSECRETMESSAGE_TLS_CERT_FILEPATH \
    SUPERSECRETMESSAGE_TLS_CERT_KEY_FILEPATH

RUN \
    apk add --no-cache ca-certificates ;\
    mkdir -p /opt/supersecret/static

WORKDIR /opt/supersecret

COPY --from=builder /go/src/github.com/pando85/crow/crow .
COPY static /opt/supersecret/static

CMD [ "./crow" ]
