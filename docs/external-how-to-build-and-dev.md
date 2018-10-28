Welcome, friend!
================

You've been linked to here from another project's "How to build" or "How to develop" -section.

The project that brought you here uses [Turbo Bob](https://github.com/function61/turbobob)
as its build (and development) tool. Turbo Bob standardizes build and development commands,
so you can use the same command in every Bob-powered project - no matter the programming
language or paradigm (desktop software, web based software, embedded software etc.).

The only prerequisite is that you have to
[install Turbo Bob](https://github.com/function61/turbobob#install). Don't worry, it's just
one self-contained binary (and it's totally open source).


How to build the project
------------------------

Just run in your project's directory:

```
$ bob build
```

To get help, run:

```
$ bob build --help
```

In more complex projects there might be arguments that you have to give to the build
command, but Bob will let your know if you missed something. The plain `build` command is
always a good start and Bob will help you with the rest.


How to develop the project
--------------------------

Just run in your project's directory:

```
$ bob dev
```

To get help, run:

```
$ bob dev --help
```

The same applies for `dev` command as for `build` - more complex projects could require
arguments, but Bob will help you if you missed something. The whole point of Bob is to
remove the misery from build tooling use and installation.


Learn more
----------

Learn more at [Bob's homepage](https://github.com/function61/turbobob).
