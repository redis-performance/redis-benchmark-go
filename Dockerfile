FROM golang:1.18 as builder
WORKDIR /work
ADD . .
RUN CGO_ENABLED=0 go build -o /redis-benchmark-go .

# runner
FROM golang:1.18-alpine
WORKDIR /
COPY --from=builder /redis-benchmark-go /bin/redis-benchmark-go

CMD ["/bin/redis-benchmark-go"
]