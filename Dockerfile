FROM alpine:latest

# "amd64" | "arm" | ...
ARG TARGETARCH
# usually empty. for "linux/arm/v7" => "v7"
ARG TARGETVARIANT

ENTRYPOINT ["/usr/bin/bob"]

COPY "rel/bob_linux-$TARGETARCH" /usr/bin/bob
