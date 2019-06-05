FROM golang:alpine

WORKDIR /go/src/github.com/PolarGeospatialCenter/pgcboot
ENV BINARY distroserver

RUN apk add --no-cache git
COPY  go.mod go.sum ./
COPY cmd ./cmd
COPY pkg ./pkg
ENV GO111MODULE on
RUN go build -o /bin/$BINARY ./cmd/$BINARY


FROM alpine:latest
ENV SSH_KNOWN_HOSTS=/root/known_hosts

RUN apk add --no-cache ca-certificates
ADD https://github.com/coreos/container-linux-config-transpiler/releases/download/v0.9.0/ct-v0.9.0-x86_64-unknown-linux-gnu /usr/bin/ct
COPY assets/known_hosts $SSH_KNOWN_HOSTS
RUN chmod a+x /usr/bin/ct

COPY --from=0 /bin/$BINARY /bin/$BINARY
CMD ["/bin/distroserver"]
