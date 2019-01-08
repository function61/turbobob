FROM fn61/buildkit-golang:20190108_1759_870854d1d97c4181

WORKDIR /go/src/github.com/function61/turbobob

CMD bin/build.sh
