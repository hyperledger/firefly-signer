[![codecov](https://codecov.io/gh/hyperledger/firefly-signer/branch/main/graph/badge.svg?token=OEI8A08P0R)](https://codecov.io/gh/hyperledger/firefly-signer)
[![Go Reference](https://pkg.go.dev/badge/github.com/hyperledger/firefly-signer.svg)](https://pkg.go.dev/github.com/hyperledger/firefly-signer)

# Hyperledger FireFly Signer

A set of Ethereum transaction signing utilities, including a Keystore V3 wallet implementation
and a runtime JSON/RPC proxy to intercept `eth_sendTransaction` JSON/RPC calls (with both
HTTPS and WebSocket support).

# License

Apache 2.0

# References / credits

### JSON/RPC proxy

The JSON/RPC proxy code was contributed by Kaleido, Inc.

### Cryptography

secp256k1 cryptography libraries are provided by btcsuite (ISC Licensed):

https://pkg.go.dev/github.com/btcsuite/btcd/btcec

### RLP encoding and keystore

Reference during implementation was made to the web3j implementation of Ethereum
RLP encoding, and Keystore V3 wallet files (Apache 2.0 licensed):

https://github.com/web3j/web3j

