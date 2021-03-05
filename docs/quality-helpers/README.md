Quality helpers
===============

Suppose you have many projects.
Bob helps you maintain standardized quality across all of your projects by having the ability to
warn you of quality/consistency issues in all your projects.


Checks
------

These are checked when you enter the dev shell (`$ bob dev`) - it's a natural point to nag the maintainers
while not breaking build process.


### Missing files check

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
		"files_that_should_exist": [
			"README.md",
			"LICENSE",
			"docs/security.md"
		]
	}
}
```

TODO: you might not be the maintainer to every project that you'd use with Bob when you have
`project_quality` settings defined.
This will cause unnecessary nags for a small fraction of Bob users - the ones that both:

- Have `project_quality` in their user config AND
- Access `$ bob dev` of a project they're not a maintainer in (& that have these files missing)

We could fix this by opting in nags for only projects that you're a maintainer in.
This could be done with a pattern like `repos from github.com/function61/*`, but we'll leave that for
later if this becomes an issue.
