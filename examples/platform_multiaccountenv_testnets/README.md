## platform_multiaccountenv_testnets

This module creates a shared Besu QBFT and IPFS testnet across three different accounts and environments w/in a Kaleido
platform instance.

The originator account creates the testnets and the joiner accounts join the testnets via
`Platform` network connectors. Joiner one requests to connect with the originator. The originator accepts the joiner
one request, and requests joiner two to connect. Joiner two accepts the request.

This represents how to create a shared network across multiple accounts and environments in a Kaleido platform instance
for software or consortium use cases.

**NOTE**: network connectors are experimental and must be enabled on your Kaleido platform instance. Please contact
support if you would like to use this feature.