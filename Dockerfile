FROM golang:1.12-alpine as builder
RUN apk update && apk add make git
WORKDIR /build
COPY . ./
RUN make dist

FROM scratch
WORKDIR /
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/meetjestad-monitor-linux /meetjestad-monitor
ENTRYPOINT ["/meetjestad-monitor"]
