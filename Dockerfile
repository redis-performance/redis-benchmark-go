FROM golang:latest AS builder
COPY . /
WORKDIR /
RUN make

FROM ubuntu:22.04
LABEL Description="redis-benchmark-go"

# Install basic utilities that might be needed
RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    bash \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /redis-benchmark-go /usr/local/bin/redis-benchmark-go

# Create a flexible entrypoint script
RUN echo '#!/bin/bash\n\
# Flexible entrypoint script\n\
if [ "$1" = "/bin/bash" ] || [ "$1" = "bash" ] || [ "$1" = "/bin/sh" ] || [ "$1" = "sh" ]; then\n\
    # If the first argument is a shell, execute it directly\n\
    exec "$@"\n\
elif [ "$#" -eq 0 ] || [ "$1" = "--help" ]; then\n\
    # Default behavior - run redis-benchmark-go with help\n\
    exec redis-benchmark-go --help\n\
else\n\
    # Otherwise, assume arguments are for redis-benchmark-go\n\
    exec redis-benchmark-go "$@"\n\
fi' > /usr/local/bin/entrypoint.sh && \
    chmod +x /usr/local/bin/entrypoint.sh

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["--help"]
