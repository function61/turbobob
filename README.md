![](misc/mascot/mascot.png)

[![Build Status](https://travis-ci.org/function61/turbobob.svg?branch=master)](https://travis-ci.org/function61/turbobob)
[![Download](https://api.bintray.com/packages/function61/turbobob/main/images/download.svg)](https://bintray.com/function61/turbobob/main/_latestVersion#files)

What is this?
-------------

Turbo Bob (the builder) is an abstraction for building and developing your software, whether it happens in your laptop or in a CI system.

Usage of Turbo Bob, in every project you're developing:

```
$ bob dev
```

This gives you a shell inside the build environment container with the working directory mounted inside the container so that you can directly edit your code files from your host system.

To build your project:

```
$ bob build
```

By keeping these commands consistent across each project we'll minimize mental friction when doing context switching, since the commands are the same for each project whether you're building a Docker-based image or running anything custom that produces build artefacts or anything custom.


Examples
--------

This very project is built with Bob on Travis. [Travis configuration](.travis.yml) is minimal - it basically just requires Docker, downloads Bob and copies `TRAVIS_COMMIT` ENV variable to `CI_REVISION_ID` and asks Bob to do the rest.

The process is exactly the same whether you use a different CI system. You can even run builds exactly the same way on your laptop by just running `$ bob build`.

Here's what happens when a new commit lands in this repo:

- Github notifies Travis of a new commit
- Travis reads [.travis.yml](.travis.yml) which downloads Bob and hands off build process to it
- Bob reads [turbobob.json](turbobob.json)
- `turbobob.json` tells Bob to build [Dockerfile.default-build](Dockerfile.default-build)
- Bob starts container based off built image of `Dockerfile.default-build` and runs [bin/build.sh](bin/build.sh) *inside the container*


Install
-------

```
$ VERSION_TO_DOWNLOAD="..." # find this from Bintray. Looks like: 20180828_1449_b9d7759cf80f0b4a
$ sudo curl --location --fail --output /usr/local/bin/bob "https://dl.bintray.com/function61/turbobob/$VERSION_TO_DOWNLOAD/bob_linux-amd64" && sudo chmod +x /usr/local/bin/bob
```
