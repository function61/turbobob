#!/bin/bash -eu

# Download Bob and hand off the build process to it

curl --fail --location --output bob https://dl.bintray.com/function61/turbobob/20180828_1449_b9d7759cf80f0b4a/bob_linux-amd64

chmod +x bob

./bob build --publish-artefacts
