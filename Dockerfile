FROM alpine:latest

ENTRYPOINT ["/usr/bin/bob"]

ADD rel/bob_linux-amd64 /usr/bin/bob
