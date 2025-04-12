FROM alpine:latest AS builder

RUN apk add --no-cache curl

RUN curl --fail --location --no-progress-meter --output /usr/local/bin/bob https://function61.com/go/turbobob-latest-stable-linux-amd64 && chmod +x /usr/local/bin/bob


FROM alpine:latest

# git = used by GitHub codespaces
# docker-cli = needed by Bob to launch sibling build containers from this "main container"
RUN apk add --no-cache git docker-cli

COPY --from=builder /usr/local/bin/bob /usr/local/bin/bob

COPY init-bob-devcontainer.sh /usr/bin/init-bob-devcontainer.sh

COPY vscode-resources/launch.json vscode-resources/tasks.json /
