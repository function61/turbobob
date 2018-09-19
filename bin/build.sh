#!/bin/bash -eu

run() {
	fn="$1"

	echo "# $fn"

	"$fn"
}

downloadDependencies() {
	dep ensure
}

checkFormatting() {
	# unfortunately we need to list formattable directories because "." would include vendor/
	local offenders=$(gofmt -l cmd/)

	if [ ! -z "$offenders" ]; then
		>&2 echo "formatting errors: $offenders"
		exit 1
	fi
}

unitTests() {
	go test ./...
}

staticAnalysis() {
	go vet ./...
}

buildLinuxArm() {
	(cd cmd/bob && GOOS=linux GOARCH=arm go build -ldflags "-X main.version=$FRIENDLY_REV_ID" -o ../../rel/bob_linux-arm)
}

buildLinuxAmd64() {
	# compile statically so this works on Alpine that doesn't have glibc
	(cd cmd/bob && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X main.version=$FRIENDLY_REV_ID -extldflags \"-static\"" -o ../../rel/bob_linux-amd64)
}

uploadBuildArtefacts() {
	# the CLI breaks automation unless opt-out..
	export JFROG_CLI_OFFER_CONFIG=false

	jfrog-cli bt upload \
		"--user=joonas" \
		"--key=$BINTRAY_APIKEY" \
		--publish=true \
		'rel/*' \
		"function61/turbobob/main/$FRIENDLY_REV_ID" \
		"$FRIENDLY_REV_ID/"
}

rm -rf rel
mkdir rel

run downloadDependencies

run checkFormatting

run staticAnalysis

run unitTests

run buildLinuxArm

run buildLinuxAmd64

if [ "${PUBLISH_ARTEFACTS:-''}" = "true" ]; then
	run uploadBuildArtefacts
fi

