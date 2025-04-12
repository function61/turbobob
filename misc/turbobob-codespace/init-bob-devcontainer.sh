#!/bin/sh -eu

# Bob maps this from host-side to new containers it launches so we must copy it to host
cp /usr/local/bin/bob /host/usr/local/bin/bob

# Bob integration to VSCode's launch config
mkdir .vscode
cp /launch.json /tasks.json .vscode/

# Bob writes stuff under /tmp which it then maps to inside Bob-spawned build containers
mount --bind /host/tmp /tmp
