FROM golang:1.19-buster as builder
WORKDIR /go/src/github.com/jaypipes/ghw

ENV GOPROXY=direct

# go.mod and go.sum go into their own layers.
COPY go.mod .
COPY go.sum .

# This ensures `go mod download` happens only when go.mod and go.sum change.
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o ghwc ./cmd/ghwc/

FROM alpine:3.20@sha256:b89d9c93e9ed3597455c90a0b88a8bbb5cb7188438f70953fede212a0c4394e0
RUN apk add --no-cache ethtool

WORKDIR /bin

COPY --from=builder /go/src/github.com/jaypipes/ghw/ghwc /bin

CMD ghwc
