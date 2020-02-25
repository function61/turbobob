Dev pro-tips
============

We have two things that cause Bob to display dev pro-tips when entering dev environment:

- Mapped network ports in dev container
- Custom dev pro-tips

We'll explain both of these along with an example `turbobob.json`


Network ports
-------------

Let's say I have an app that has a network server (like HTTP server) and I want to map its
port (80) to be visible from the host port (8084).


Dev pro-tips
------------

I have a static blog generator where there is a preview step before I run the actual build.

When coming back to writing a blog post from a long break, it's hard to remember what the
preview command was.

For general pro-tips, Bob has a mechanism for displaying pro-tips when you enter the
dev environment (= the container).


How pro-tips look when I enter the container
--------------------------------------------

Given below example file.. on entering the container:

```console
$ bob dev
Pro-tip: mapped dev ports: 8084:80
Pro-tip: For preview run $ bin/preview.sh
$ 
```

Example turbobob.json
---------------------

This example demoes both network ports and and custom "dev pro-tips":

```json

{
	"for_description_of_this_file_see": "https://github.com/function61/turbobob",
	"version_major": 1,
	"project_name": "joonas.fi-blog",
	"builders": [
		{
			"name": "default",
			"uses": "docker://joonas/jekyll-builder:0.1.0",
			"mount_destination": "/project",
			"workdir": "/project",
			"commands": {
				"build": ["bin/build.sh"],
				"dev": ["bash"]
			},
			"dev_ports": ["8084:80"],
			"dev_pro_tips": ["For preview run $ bin/preview.sh"]
		}
	]
}
```

The above makes Docker map host port `8084` -> container port `80`.

