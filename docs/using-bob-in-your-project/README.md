Using Bob in your project
=========================

The best way to explain this is to start from scratch - from an empty directory.

Our goal:

- Get `$ bob build` to build a Go-based application (don't worry - knowledge of Go is not required)
- Get the same working in a CI system (we'll use Travis CI)

Let's get started!

Contents:

- [Determine which Docker image to use to build your app](#determine-which-docker-image-to-use-to-build-your-app)
- [Create turbobob.json](#create-turbobobjson)
- [Add code that you want to compile](#add-code-that-you-want-to-compile)
- [Compile the code manually inside the container](#compile-the-code-manually-inside-the-container)
- [Have Bob compile your program](#have-bob-compile-your-program)
- [Making it build in Travis CI](#making-it-build-in-travis-ci)
- [(optional) Next steps](#optional-next-steps)
- [(optional exercice) Creating your own builder image](#optional-exercice-creating-your-own-builder-image)


Determine which Docker image to use to build your app
-----------------------------------------------------

If you want to get knee-deep in container image creation from scratch, there's a bonus
chapter at the end of this guide. But to hit the ground running, let's start with a
ready-made image.

The image (and version) we want to use for builds is `fn61/buildkit-golang:20200212_0907_06f93bc3`.


Create turbobob.json
--------------------

Now that we know the image we want to use for builds, write this `turbobob.json`:

```json
{
	"for_description_of_this_file_see": "https://github.com/function61/turbobob",
	"version_major": 1,
	"project_name": "helloworld",
	"builders": [
		{
			"name": "default",
			"uses": "docker://fn61/buildkit-golang:20200212_0907_06f93bc3",
			"mount_destination": "/workspace",
			"workdir": "/workspace",
			"commands": {
				"build": ["go", "build", "-o", "rel/hello"],
				"dev": ["bash"]
			}
		}
	],
	"os_arches": {
		"linux-amd64": true
	}
}
```

Notes for the curious:

- The builder name `default` has no other function than when you say `$ bob dev` it defaults
to that one unless you specify some other builder with `$ bob dev someOtherBuilder`.

- `os_arches` are flags to which architectures we wish to compile our project.
  `buildkit-golang` supports cross-compiling to many OSes/architectures. Not all buildkits
  need/use these flags (for example JavaScript builders don't have concept or architectures or OSes).

- The rest are explained later.

Your directory should now look like this:

```
helloworld/
└── turbobob.json
```


Add code that you want to compile
---------------------------------

Let's add a simple Go code as `main.go`:

```go
package main

import (
	"fmt"
)

func main() {
	fmt.Println("Hello world")
}
```

Your directory should now look like this:

```
helloworld/
├── main.go
└── turbobob.json
```


Compile the code manually inside the container
----------------------------------------------

To enter the container, all you have to do is run `$ bob dev`. You can identify you're
inside a container, when your prompt looks like this:

```
root@64cd792a5d0e:/workspace# 
```

(Note: you can exit container by pressing `ctrl + d` or by typing `exit`.)

Remember in `turbobob.json` where we had a line that had
`"build": ["go", "build", "-o", "rel/hello"]`?

That's the build command that Bob would run inside the container. You'll see it next, when
we run it manually from Bash. Now let's hop into the container and compile it:

```console
$ bob dev
$ go build -o rel/hello
$ # Let's also run the program to test it works
$ rel/hello
Hello world
```

It should build successfully. Your directory should now look like this:

```
helloworld/
├── main.go
├── rel
│   └── hello
└── turbobob.json
```

Exit the container.


Have Bob compile your program
-----------------------------

Now that you managed to compile the program manually from inside the container, Bob should
be able to compile it as well.

Remove the compiled binary first with `$ rm rel/hello` so we can see verify that Bob can
build it.

Bob requires version control. In this tutorial version control is not our concern, but we
have to initialize a repository and at least one commit:

```console
$ git init
$ git add main.go turbobob.json
$ git commit -m "added initial files"
```

Now run:

```console
$ bob build --uncommitted
```

(Note: `--uncommitted` switch because otherwise Bob would build the latest commit but in
another directory. It should succeed but the build artefacts would appear in
`/tmp/bob/helloworld/workspace/rel/hello` or similar)

The build should have now succeeded.


Making it build in Travis CI
----------------------------

To integrate with Travis, we need a config file for Travis.

Bob has a neat trick of writing config file boilerplates for the most common CI systems -
you can run:

```console
$ bob tools init --travis
cannot init; Bobfile already exists
```

You'll get that error because you already have `turbobob.json` in this directory - the init
command is meant to generate the `turbobob.json` (we already have made one).

But the requested `travis.yml` file was still written anyway. The file is usable as-is -
you don't have to make modifications.

Now your directory should look like this:

```
helloworld/
├── main.go
├── rel
│   └── hello
├── .travis.yml
└── turbobob.json
```

Setup repo integration in Travis CI, commit your `.travis.yml`, push to GitHub and your
Travis build should kick off and eventually succeed. Nice work!


(optional) Next steps
---------------------

The `buildkit-golang` buildkit that we used, has much more to offer like:

- checking that code is formatted properly
- running static analysis
- running tests
- packaging the binary as AWS Lambda function

Also, you might want to do add build artefact publishing (upload built binaries to GitHub
releases or Bintray) to your workflow.

These things are out of scope of this tutorial, but you can study the example projects'
`turbobob.json` files that you can find from our main README.


(optional exercice) Creating your own builder image
---------------------------------------------------

Ok, let's learn how to make our own builder.

The `buildkit-golang` one that we used is built on top of
[Go's official Docker image](https://hub.docker.com/_/golang). We could use that but then
we'd be done right away - it already contains the Go compiler.

So let's start with an Ubuntu image that doesn't already contain the Go compiler. Let's
prototype inside an Ubuntu container:

```console
$ docker run --rm -it ubuntu bash
$ go version
bash: go: not found
$ # so yes, Go compiler does not exist
$ apt update && apt install -y golang
$ go version
go version go1.13.4 linux/amd64
$ # so, now we know the command to install Go on top of Ubuntu
```

Let's codify that as a Docker image. Write into `builder-default.Dockerfile` this:

```
FROM ubuntu

RUN apt update && apt install -y golang
```

Let's test that it builds with Docker:

```console
$ docker build --file builder-default.Dockerfile --tag my-first-builder .
```

It should succeed.

Your directory should now look like this:

```
helloworld/
├── builder-default.Dockerfile
├── main.go
├── rel
│   └── hello
└── turbobob.json
```

Now just change `turbobob.json` to use your builder image instead of the `buildkit-golang`
one:

```diff
        "builders": [
                {
                        "name": "default",
-                       "uses": "docker://fn61/buildkit-golang:20200212_0907_06f93bc3",
+                       "uses": "dockerfile://builder-default.Dockerfile",
                        "mount_destination": "/workspace",
                        "workdir": "/workspace",
                        "commands": {

```

Note: when using `dockerfile://` (instead of `docker://`), Bob builds the builder for you
before running the container. That way you can ship customizations to other people's
Docker images inside your own repo.

But `dockerfile://` approach is slower (on CI systems since Docker's build cache is not
warm), so if you want speed and to reuse the changes in your builder in other projects,
you should push your builder image to DockerHub or similar so you can go back to the
`docker://` syntax that uses already-made images. That's basically just one `$ docker push`
away now that your `$ docker build ...` command succeeds. :)

Now once again remove the built binary by running `$ rm rel/hello` so we can verify that
Bob builds the binary with your freshly baked builder image:

```console
$ bob build --uncommitted
```

You've now built your hello world app with a buildkit image that you built yourself! :)
