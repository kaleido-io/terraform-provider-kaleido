## platform_canton_sandbox_network

This module creates a Canton sandbox network in the Kaleido platform instances. The sandbox network mimics The Global Synchronizer network and is enabled with canton coins. 

> **NOTE**: Please contact Kaleido support to ensure you have canton enabled on your Kaleido platform.

The following resources are created
- **Networks**
    - Canton Validator Network: Network configuration to mimic The Global Synchronier network, enabled with Canton coins
    - Canton Synchronizer Network:Network configuration to create a private synchronizer. Optional, can be enabled with `enable_synchronizer_network = true`
- **Nodes**
    - Canton Super validator: Part of the Canton Validator network that manages the governance operations
    - Canton Synchronizer node: Part of the Canton Synchronizer network that orders and propogates transactions to the involved parties. Optional, can be enabled with `enable_synchronizer_network = true`
    - Canton Participant node: Connected to both the network and manages transaction submission and smart contract state for an entity on the network
- **Services**
    - KMS: Key management service to securely store and manage keys for all the canton nodes
  

