#!/bin/bash -eu

curl --fail --location --output bob https://dl.bintray.com/function61/turbobob/dev/bob_linux-amd64

chmod +x bob

./bob build
