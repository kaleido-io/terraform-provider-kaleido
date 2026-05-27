# Kaleido Terraform Provider

The Kaleido Terraform Provider is our official provider for the [Kaleido Enterprise Platform](https://docs.kaleido.io/platform/) _and_ [Kaleido Blockchain-as-a-Service](https://docs.kaleido.io/baas/). 

Check out the [Kaleido Terraform Provider documentation](https://registry.terraform.io/providers/kaleido-io/kaleido/latest/docs) for the latest provider schema and examples.

## Getting Started

**Prerequisites:**
- [OpenTofu](https://opentofu.org/docs/intro/) or [Terraform](https://developer.hashicorp.com/terraform/install)
- A Kaleido Enterprise Platform account and API key

```hcl
terraform {
  required_providers {
    kaleido = {
      source = "kaleido-io/kaleido"
      version = ">1.2.0"
    }
  }
}

provider "kaleido" {
  alias = "provider"
  platform_api = var.kaleido_platform_api                 # https://<account-name>.${PLATFORM_DOMAIN}
  platform_username = var.kaleido_platform_username       # the name of the API key
  platform_password = var.kaleido_platform_password       # the secret of the API key
}
```

See the [official Terraform modules](https://github.com/kaleido-io/terraform-kaleido-modules) repository for useful modules and relevant examples.

## Development

**Prerequisites:**
- Go 1.25 or greater
- make


### Build & Unit Tests

```sh
make
```

### Using

To install the provider from a local build with Terraform 1.x or OpenTofu, configure your `~/.terraformrc` with:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/kaleido-io/kaleido" = "/path/to/terraform-provider-kaleido" # for Terraform
    "kaleido-io/kaleido" = "/path/to/terraform-provider-kaleido"                       # for OpenTofu
  }
  direct {}
}
```

then be sure to build the binary you are testing using:

```shell
make build
```

> NOTE: binaries built via `make build-${OS}` will not be detected by Terraform's `dev_overrides`.

Kaleido Terraform Provider uses [terrraform-plugin-docs](https://github.com/hashicorp/terraform-plugin-docs) to generate all documentation markdown files. To update the provider documentation after any schema, example, or description changes run:

```shell
make docs
```

## Cross Compiling

```
make build-linux
make build-mac
make build-win
```
