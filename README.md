![](misc/mascot/mascot.png)

[![Build Status](https://img.shields.io/travis/function61/turbobob.svg?style=for-the-badge)](https://travis-ci.org/function61/turbobob)
[![Download](https://img.shields.io/bintray/v/function61/dl/turbobob.svg?style=for-the-badge&label=Download)](https://bintray.com/function61/dl/turbobob/_latestVersion#files)

Modern, minimal container-based build/development tool to make any project´s dev easy and frictionless.

![](docs/demo-screencast.gif)


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

By keeping these commands consistent across each project we'll minimize friction with
mental context switching, since the commands are the same for each project whether
you're building a Docker-based image or running anything custom that produces build
artefacts.

There's a [document that your project can link to](docs/external-how-to-build-and-dev.md)
for build & help instructions. This explains Bob's value proposition quite well and serves
as the first introduction for new Bob users. See an
[example of a project's build docs linking to Bob](https://github.com/function61/ruuvinator#how-to-build--develop).


Philosophy
----------

- Your project must support a simple `build` and `dev` interface. If you can't, you're
  probably doing something wrong and you should simplify it. The `build` command usually just
  runs your project's `bin/build.sh` (or equivalent) command inside a container. The `dev`
  command usually starts Bash terminal inside the container but doesn't execute `bin/build.sh`
  so you can manually invoke or debug the build process (or a subset of it).

- Build environment should be stateless & immutable. No longer missing build tools or
  mismatched versions within your team. Nothing to install on your CI server (except Docker).

- Decouple build-time dependencies from runtime dependencies
  ([build container pattern](https://medium.com/@alexeiled/docker-pattern-the-build-container-b0d0e86ad601)),
  so build tools will not be shipped to production (smaller images & less attack surface).

- Dev/CI/production environment parity as close as possible. Dev environment is the same as
  build & CI environment. What's built on dev (`$ bob build`) is exactly the same or as
  close as possible (`$ bob build --uncommitted`) as to what will end up running in production.

- No vendor lock-in for a CI system. Bob can seamlessly build projects on your laptop, Jenkins,
  Travis, GitLab etc. CI needs to only provide the working directory and Docker - everything
  else like uploading artefacts to S3, Bintray etc. should be a build container concern to
  provide full independence.


Install
-------

Requires Docker for use, so currently only Linux is supported. Windows support might come
later as Windows' Linux subsystem keeps maturing.

```
$ VERSION_TO_DOWNLOAD="..." # find this from Bintray. Looks like: 20180828_1449_b9d7759cf80f0b4a
$ sudo curl --location --fail --output /usr/local/bin/bob "https://dl.bintray.com/function61/dl/turbobob/$VERSION_TO_DOWNLOAD/bob_linux-amd64" && sudo chmod +x /usr/local/bin/bob
```


Examples
--------

A few sample projects that shows how Turbo Bob is used for builds:

- [function61/james](https://github.com/function61/james)
	- uses buildkit [function61/buildkit-golang](https://github.com/function61/buildkit-golang)
- [function61/lambda-alertmanager](https://github.com/function61/lambda-alertmanager)
	- uses buildkit [function61/buildkit-js](https://github.com/function61/buildkit-js)
- [function61/home-automation-hub](https://github.com/function61/home-automation-hub)
	- uses *both* Go- and JS buildkits (which neatly demoes the hygiene of keeping different
	ecosystems' build tools separate - they could even run different distros!)

What is a buildkit? It's not strictly a Turbo Bob concept - it only means that instead of
constructing the whole build environment in your own repo in the `build-default.Dockerfile`, that
Dockerfile is mostly empty and most of its configuration comes from the `FROM` image referenced
in the Dockerfile from another repo. This makes for smaller build Dockerfiles (but you can
still do customizations). This makes builds faster and increases standardization and
reusability across projects whose build environments will be similar anyways.


How does it work?
-----------------

This very project is built with Bob on Travis. [Travis configuration](.travis.yml) is minimal - it basically just:

- Requires Docker
- Downloads Turbo Bob
- Copies `TRAVIS_COMMIT` ENV variable to `CI_REVISION_ID` and
- Asks Bob to do the rest:

The process is exactly the same whether you use a different CI system. You can even run builds exactly the same way on your laptop by just running `$ bob build`.

Here's what happens when a new commit lands in this repo:

- Github notifies Travis of a new commit
- Travis reads [.travis.yml](.travis.yml) which downloads Bob and hands off build process to it
- Bob reads [turbobob.json](turbobob.json)
- `turbobob.json` tells Bob to build [build-default.Dockerfile](build-default.Dockerfile)
- Bob starts container based off built image of `build-default.Dockerfile` and runs [bin/build.sh](bin/build.sh) *inside the container*
