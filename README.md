# github_pull_trigger

Git Pull trigger can actively pull github private repo and do a post to target svc such as tekton event listener. 

This is useful when github webhook can't reach to event listener deployed in private cloud.

So this is pull strategy trigger instead push method.

The event support:
* github push event
* github PR update event

# Start in local env via podman

## Build
podman build -t github-pull-trigger:latest .

## Run 
```
podman run --rm -e GITHUB_PAT_TOKEN=xxx \
  -v ./config.json:/opt/app-root/config.json:Z \
  localhost/github-pull-trigger:latest
```

# Deploy as deployment in OpenShift

# Deploy as a sidecar for event listener
