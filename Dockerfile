FROM golang:1.11 AS builder
WORKDIR /go/src/app
COPY . .
RUN go build -o /usr/bin/prometheus-remote-text .

###############################################

FROM ubuntu:16.04
COPY --from=builder /usr/bin/prometheus-remote-text /usr/bin/prometheus-remote-text
CMD ["/usr/bin/prometheus-remote-text"]
