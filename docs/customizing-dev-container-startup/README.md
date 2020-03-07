Customizing dev container startup
=================================

Dry run
-------

`$ bob dev` runs `$ docker run ...` under the hood. There are times when you need to add
some special features to the container that would be started by `$ bob dev`. Examples:

- Adding special Linux capabilities with `--cap-add`
- Setting custom DNS server
- etc. anything special.

For these cases you can ask Bob to dump the `$ docker run ...`  command that it would invoke:

```console
$ bob dev --dry
docker run --rm --interactive --tty --name tbdev-ubackup-default --volume /vagrant/ubackup/:/workspace --workdir /workspace --env FRIENDLY_REV_ID=dev --env BUILD_LINUX_AMD64=true fn61/buildkit-golang:20200212_0907_06f93bc3 bash
```

Now you can copy the command, add your customizations and run it directly without Bob. The
end result is the same if it would be run by `$ bob dev`. You still get all the goodies
like when you run `$ bob dev` and a container already exists, Bob gives you an additional
shell inside the same container (equivalent to running `$ docker exec -it <container_id> bash`).


Upcoming features
-----------------

We should consider adding syntax like this, which probably would cover most of dry run's
use cases:

```diff
    "uses": "docker://fn61/buildkit-golang:20200212_0907_06f93bc3",
    "mount_destination": "/workspace",
    "workdir": "/workspace",
+   "dev_docker_run_extra_flags": ["--cap-add", "NET_BIND_SERVICE"],
    "commands": {
        "build": ["bin/build.sh"],
        "dev": ["bash"]
```
