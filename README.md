# AI-infra
# AI-infra

## Build

Use the root `Dockerfile` for every service image:

```sh
make docker-build
```

Image tags start at `vv1` and increment from the highest local
`promethius:vvN` or `employee-service:vvN` tag. To see the next tag without
building:

```sh
make docker-version
```
