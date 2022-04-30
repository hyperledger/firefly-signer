[![codecov](https://codecov.io/gh/hyperledger/firefly-signer/branch/main/graph/badge.svg?token=OEI8A08P0R)](https://codecov.io/gh/hyperledger/firefly-signer)
[![Go Reference](https://pkg.go.dev/badge/github.com/hyperledger/firefly-signer.svg)](https://pkg.go.dev/github.com/hyperledger/firefly-signer)

# Hyperledger FireFly Signer

A set of Ethereum transaction signing utilities designed for use across projects:

- RLP Encoding and Decoding
- Secp256k1 transaction signing for Ethereum transactions
  - Original
  - EIP-155
  - EIP-1559
- Keystore V3 wallet implementation
  - Scrypt - read/write
  - pbkdf2 - read

A runtime JSON/RPC server/proxy to intercept `eth_sendTransaction` JSON/RPC calls

- Lightweight fast-starting runtime
- HTTP/HTTPS server
  - All HTTPS/CORS etc. features from FireFly Microservice framework
  - Configured via YAML
- `eth_sendTransaction` implementation to sign transactions
  - If EIP-1559 gas price fields are specified uses `0x02` transactions, otherwise EIP-155
- Makes some JSON/RPC calls on application's behalf
  - Queries Chain ID via `net_version` on startup
  - `eth_account` support
  - Trivial nonce management built-in (calls `eth_getTransactionCount` for each request)
- File based wallet
  - Configurable caching for in-memory keys
  - Files in directory with a given extension matching `{{ADDRESS}}.key`/`{{ADDRESS}}.toml`
  - Customizable extension, and optional `0x` prefix to filename
  - Files can be TOML/YAML/JSON metadata pointing to Keystore V3 files + password files
  - Files can be Keystore V3 files directly, with accompanying `{{ADDRESS}}.pass` files

Potential future contributions:

- WebSockets support
- Tessera private transaction signing for Quorum / Hyperledger Besu
- Loading keys on startup
- Caching list of keys in-memory
- Regular expression to match address anywhere in filename (depends on caching list of keys)

# Example configuration

Two examples provided below:

## Flat directory of keys

```yaml
fileWallet:
    path: /data/keystore
    filenames:
        with0xPrefix: false
        primaryExt: '.key.json'
        passwordExt: '.password'
server:
    address: '127.0.0.1'
    port: 8545
backend:
    url: https://blockhain.rpc.endpoint/path
```

## Directory containing TOML configurations

```yaml
fileWallet:
    path: /data/keystore
    filenames:
        with0xPrefix: false
        primaryExt: '.toml'
  metadata:
        format: toml
        keyFileProperty: '{{ index .signing "key-file" }}'
        passwordFileProperty: '{{ index .signing "password-file" }}'
server:
    address: '127.0.0.1'
    port: 8545
backend:
    url: https://blockhain.rpc.endpoint/path
```

Example TOML:

```toml
[metadata]
description = "File based configuration"

[signing]
type = "file-based-signer"
key-file = "/data/keystore/1f185718734552d08278aa70f804580bab5fd2b4.key.json"
password-file = "/data/keystore/1f185718734552d08278aa70f804580bab5fd2b4.pwd"

```

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

