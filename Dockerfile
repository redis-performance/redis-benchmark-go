FROM golang:latest AS builder
COPY . /
WORKDIR /
RUN make

FROM ubuntu:24.10
LABEL Description="redis-benchmark-go"
COPY --from=builder /redis-benchmark-go /usr/local/bin

ENTRYPOINT ["redis-benchmark-go"]
CMD [ "--help" ]
