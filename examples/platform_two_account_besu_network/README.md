## platform_two_account_besu_network

This module creates a shared Besu QBFT testnet across two different accounts in one or more Kaleido platform instances.
It leverages `Permitted` network connectors to allow the  node identities and connectivity endpoints to be trusted between
the two parties. It assumes the zones being used are enabled to a network connectivity plugin that allows for external
IP addresses to be attached to node runtimes.

**NOTE**: please contact Kaleido support to ensure you have runtimes zones with such network connectivity enabled. If
you are running the Kaleido platform instance as software, please consult the software documentation for how you can
configure, and even build, your own network connectivity plugins.

This represents how to create a shared network across multiple accounts in different Kaleido platform instances, possibly
in different regions or cloud providers.

**NOTE**: network connectors are experimental and must be enabled on your Kaleido platform instance. Please contact
support if you would like to use this feature.