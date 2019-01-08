FROM fn61/buildkit-golang:20190108_1812_e64c80f1

WORKDIR /go/src/github.com/function61/turbobob

CMD bin/build.sh
