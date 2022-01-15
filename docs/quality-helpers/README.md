Quality helpers
===============

Suppose you have many projects.
Bob helps you maintain standardized quality across all of your projects by having the ability to
warn you of quality/consistency issues in all your projects.


Checks
------

These are checked when you enter the dev shell (`$ bob dev`) - it's a natural point to nag the maintainers
while not breaking build process.


### File must exist -check

If you're missing "important ingredients", like:

- Having a main README
- Having a license declared
- Having a security policy
	* This (like the others above)
	  [will be shown automatically by GitHub's UI](https://github.com/function61/varasto/security/policy)

All of the above can be accomplished with a "file exists" check. This is done by adding to Bob's
user-specific configuration file:

```json
{
	"project_quality": {
		"file_rules": [
			{
				"path": "README.md",
				"must_exist": true
			},
			{
				"path": "LICENSE",
				"must_exist": true
			},
			{
				"path": "docs/security.md",
				"must_exist": true
			}
		]
	}
}
```

TODO: you might not be the maintainer to every project that you'd use with Bob when you have
`project_quality` settings defined.
This will cause unnecessary nags for a small fraction of Bob users - the ones that both:

- Have `project_quality` in their user config AND
- Access `$ bob dev` of a project they're not a maintainer of (& that have these files missing)

We could fix this by opting in to nags from only projects that you're a maintainer of.
This could be done with a pattern like `repos from github.com/function61/*`, but we'll leave that for
later if this becomes an issue.


### File must not exist -check

Now let's say that you learned that GitHub can
[specify security policy organisation-wide](https://docs.github.com/en/communities/setting-up-your-project-for-healthy-contributions/creating-a-default-community-health-file),
and now you're left with deleting repository-specific security policies because you want for the
organization-wide policy to be inherited to GitHub's UI.

You can model this as a file must not exist check:

```json
{
	"project_quality": {
		"file_rules": [
			{
				"path": "docs/security.md",
				"must_exist": false
			}
		]
	}
}
```


### File content contains -check

Let's say you want to guarantee that each of your project README has installation instructions header:

```json
{
	"project_quality": {
		"file_rules": [
			{
				"path": "README.md",
				"must_contain": [
					"## Installation"
				]
			}
		]
	}
}
```

Note: the check is only done if `README.md` exists. If you want to ensure that the file exists, combine it with `"must_exist": true` rule:

```json
{
	"project_quality": {
		"file_rules": [
			{
				"path": "README.md",
				"must_exist": true,
				"must_contain": [
					"## Installation"
				]
			}
		]
	}
}
```


### File content does not contain -check

Same as contains, but use `must_not_contain`.

Use case: you've copy-pasted content into many projects and now you've to remember to remove that
from each project.


### Ensure you update to latest version of a builder

Let's say you have 20 projects that use the same [buildkit-golang](https://github.com/function61/buildkit-golang)

To not break builds/dev process, you don't simply YOLO by using `fn61/buildkit-golang:latest` in
your projects' `turbobob.json`. Rather, you pin version like `fn61/buildkit-golang:20210702_0854_7adda4a2`.

Updating (and thus possibly breaking your build) should be an explicit decision of the maintainer,
but we'd still like to ensure we don't forget to update to the latest version when you come back to a
specific project that's not been touched for a while.

You can write a rule like this:

```json
{
	"project_quality": {
		"builder_uses_expect": {
			"docker://fn61/buildkit-golang": "docker://fn61/buildkit-golang:20210702_0854_7adda4a2"
		}
	}
}
```

Rule `builder_uses_expect` works by simple `substring => entire expected string` mappings.
I.e. given this (partial) `turbobob.json` ..:

```json
{
	"builders": [
		{
			"name": "default",
			"uses": "docker://fn61/buildkit-golang:20210702_0854_7adda4a2"
		},
		{
			"name": "publisher",
			"uses": "docker://fn61/buildkit-publisher:20200228_1755_83c203ff"
		}
	]
}
```

.. the rule would check the default builder (but not publisher) because the default builder's `uses`
contains the substring `docker://fn61/buildkit-golang`. It then checks that the entire string should
equal `docker://fn61/buildkit-golang:20210702_0854_7adda4a2`.
If it does not, you'll get a nag that you're running an outdated builder image.