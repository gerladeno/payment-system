FROM golang:1.16-alpine3.13 as builder

ADD . /src/app
WORKDIR /src/app
RUN go mod download

ARG VERSION
RUN GOOS=linux CGO_ENABLED=0 go build -ldflags " -X main.version=${VERSION}" -o payments ./cmd/payments/

FROM alpine:3.13

COPY --from=builder /src/app/payments /payments

ENTRYPOINT ["/payments"]