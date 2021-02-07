Language server support
=======================

People like nice things:

- Code completion
- Analysis about problems
- Refactoring (rename a variable/function/etc everywhere where it's used)
- Hints for function parameters

A [language server](https://langserver.org/) is a modern way to get these in your favourite code editor.


Why a language server?
----------------------

Before language servers, each code editor had to have code for the mentioned features for:

- C code
- Go code
- JavaScript code
- CSS code
- HTML
- ...

It looked like this:

![](lsp-problem.png)

Now with a language server, the logic that used to be implemented inside each
code editor separately for a given programming language can live inside a single language-specific
server, and all the code editors can use that one server to parse information for that programming language.

A code editor needs to only implement the Language Server Protocol ("LSP") standard to gain access
to all the programming languages - because you now know how to talk each specific language's language server:

![](lsp-solution.png)

For Go, its language server is named [gopls](https://github.com/golang/tools/tree/master/gopls).
Usually a code editor starts the LS as its child process and communicates with it using stdin/stdout -
but network (TCP/IP) is also widely supported.

Let's discuss LSP & Bob. We're going to use `gopls` as example of a specific LS.


Bob's model
-----------

Bob likes for all build tooling to be inside containers. There are many advantages:

- You don't have to install anything in your host system
- The entire build environment is easily shippable (and thus usable!) inside one well-defined container
  * This preferably includes the LS for the specific language


How language servers are usually used
-------------------------------------

This is how LS's are "traditionally" used:

![](lsp-traditionally.png)


Bob & LSP
---------

So, Bob likes containers and thus we need `gopls` to be in a container. Therefore the process
now has to look like this:

![](lsp-in-container.png)

Because traditionally the code editor and the language server is expected to be in the same
computer (or to be more exact with containers: the same namespace), there usually is a difference
in the expectations of default usage behaviour and how you want to use it with Bob. Specifically:

| issue | Traditional approach | Bob's way |
|-------------------------|---|
| Start a language server | An editor starts `gopls` a as child process | We want to start `gopls` inside a container |
| Multiple LS instances? | An editor expects one `gopls` instance can access any Go-based project | Each project (also the language server) is in own container, so it can only access the chosen project's files |

To help bridge these differences, Bob has a small shim (`$ bob lsp`) for bridging these differences.


Under construction
------------------

The LSP support is in its early stages.


### Here be dragons

Bob's LSP shim isn't perfect yet, for example depending on your editor you can only access one
Go-based project at a time. At least Sublime Text expects that if `gopls` is running, it can use that
instance to access projects A and B, but the `gopls` is in A's container (assuming that project was
opened first) and thus can't access B's files.

The following has been tested to work:

- Bob + Sublime Text + gopls


### Install instructions

[Install sublimelsp into Sublime Text](https://github.com/sublimelsp/LSP) to first get LSP support.

Edit LSP (user-specific) settings to contain:

```json
{
	"clients": {
		"gopls": {
			"command": ["bob", "lsp"],
			"enabled": true
		}
	}
}
```

Enter your dev container first (`$ bob dev`)! The container has to be based on `fn61/buildkit-golang`.

Now you can `LSP: Enable Language Server in Project` and open a `.go` file!
