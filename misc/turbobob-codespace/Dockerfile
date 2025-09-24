FROM alpine:latest AS builder

RUN apk add --no-cache curl

RUN curl --fail --location --no-progress-meter --output /usr/local/bin/bob https://function61.com/go/turbobob-latest-stable-linux-amd64 && chmod +x /usr/local/bin/bob


FROM alpine:latest

# git = used by GitHub codespaces
# docker-cli = needed by Bob to launch sibling build containers from this "main container"
# docker-cli-buildx = Docker's buildx plugin
# bash = a bit more familiar shell for the user
# nano = rudimentary shell-based editor
# shadow = required to call `$ chsh` (wich is used to change user's default shell)
RUN apk add --no-cache git docker-cli docker-cli-buildx bash nano shadow \
  && chsh -s /bin/bash \
  && echo 'cat /usr/local/etc/vscode-dev-containers/first-run-notice.txt' > /root/.bashrc

# COPY --from=builder /usr/local/bin/bob /usr/local/bin/bob
COPY bob_linux-amd64 /usr/local/bin/bob

COPY init-bob-devcontainer.sh /usr/bin/init-bob-devcontainer.sh

COPY vscode-resources/launch.json vscode-resources/tasks.json /

COPY first-run-notice.txt /usr/local/etc/vscode-dev-containers/

