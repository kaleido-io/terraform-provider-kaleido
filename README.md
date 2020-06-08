# Kaleido Terraform Provider

## Build & Unit Tests

```sh
make
```

## Using

End to end example in [examples/simple_env](examples/simple_env)

Additional examples for different Kaleido REST resource types are in the unit/acceptance tests, for example:
[kaleido/resource_service_test.go](kaleido/resource_service_test.go#L108)

## Cross Compiling

```
make build-linux
make build-mac
make build-win
```

## Acceptance Tests

Acceptance tests make actual calls to deploy and destroy resources.
Any changes to the provider must pass acceptance tests.

```sh
export TF_ACC=true
export KALEIDO_API='https://control-stage.kaleido.io/api/v1'
export KALEIDO_API_KEY=XXXXXXX=
go test -v ./kaleido
```

> Note unit tests are now being prioritized and the process of migrating acceptance tests to
> unit tests has been started in [kaleido_service_test.go](./kaleido/kaleido_service_test.go)

