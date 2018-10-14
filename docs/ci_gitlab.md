Bob for GitLab uses GitLab.com's (the cloud hosting)
[shared runners](https://docs.gitlab.com/ee/ci/runners/).

They are pretty tricky because they force us to build inside a container. Inside a container
we normally can't invoke Docker (which is what Bob needs), but there's a special image we
use, "dind" (Docker-in-Docker), which seems like a hack but it works.

Links:

- https://gitlab.com/ayufan/container-registry/blob/master/.gitlab-ci.yml
- https://gitlab.com/gitlab-org/gitlab-runner/issues/1250
- https://stackoverflow.com/questions/39608736/docker-in-docker-with-gitlab-shared-runner-for-building-and-pushing-docker-image
