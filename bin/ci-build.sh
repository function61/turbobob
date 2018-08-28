#!/bin/bash -eu

curl --fail --location --output bob https://dl.bintray.com/function61/turbobob/20180828_1241_08924ed6611f4520/bob_linux-amd64

chmod +x bob

./bob build
