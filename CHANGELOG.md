# CHANGELOG

## v1.2.0

- OpenTofu support
- New resources:
  - `kaleido_platform_account`
  - `kaleido_platform_user`
  - `kaleido_platform_group_membership`
  - `kaleido_platform_besu_node_key`
- Importable resources:
  - `kaleido_platform_account`
  - `kaleido_platform_user`
- Additional examples:
 - TODO

## v1.1.2

- Bug fix for updating `kaleido_platform_kms_wallet` and wallet credentials

## v1.1.1

- Improved documentation and examples
- `force_delete` option on `kaleido_platform_services`
- Asset manager resource enhancements
- CVE fixes and improved scanning
- New `kaleido_network_connector` resource
- New resources allowing for support of Stacks, Applications, access, and remote node peering:
 - `kaleido_platform_stack`
 - `kaleido_network_connector`
 - `kaleido_platform_application`
 - `kaleido_platform_service_access`
 - `kaleido_platform_stack_access`

## v1.1.0

- Rewritten to be based on the `terraform-plugin-framework`
- Introduces `kaleido_platform*` for the next generation Kaleido Platform

## v1.0.x

- Initial Terraform provider for the Kaleido Blockchain-as-a-Service (BaaS)
