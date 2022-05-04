# Kaleido Terraform Provider

## Build & Unit Tests

```sh
make
```

## Using

To install the provider from a local build with Terraform 0.14, configure your `~/.terraformrc` with:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/kaleido-io/kaleido" = "/path/to/terraform-provider-kaleido"
  }
  direct {}
}
```

then be sure to build the binary you are testing using:

```shell
make build
```

> NOTE: binaries built via `make build-${OS}` will not be detected by Terraform's `dev_overrides`.

## Examples

End to end example in [examples/multi_region_with_b2b](examples/multi_region_with_b2b)

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

