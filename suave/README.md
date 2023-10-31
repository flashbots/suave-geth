# Batch auction suave application

This illustrates the "on-chain intents" pattern in SUAVE.
Intents are collected on SUAVE chain in an encrypted form.
At any time we can "solve" a batch of intents, sending a bundle to EthL1.

The guarantee provided here is that solvers are constrained to processing the SUAVE on-chain batch, all or nothing. Kettle nodes cannot censor any encrypted message posted on SUAVE chain.

## To run
```
forge script ./sol/scripts/batchauction_forge.sol --ffi --tc EncryptionExample
```

## Code layout

- [sol/scripts/batchauction_forge.sol](sol/scripts/batchauction_forge.sol)

    Example scenarios
- [sol/batchauction.sol](sol/batchauction.sol)
  
    This includes both the EthL1 contract as well as the Suave chain contract. The EthL1 contract receives batches of fulfilled orders at a time.

    Encrypted orders are submitted on the SUAVE chain contract. The encrypted payload in this case is just a message. Solvers are constrained to invoke the `completeBatch` function and that's it.
- [sol/libraries/ethtransaction.sol](sol/libraries/ethtransactions.sol) and [sol/libraries/RLPEncode.sol](sol/libraries/RLPEncode.sol)

    For constructing an Ethereum transaction within SUAVE.
- [sol/libraries/encryption.sol](sol/libraries/encryption.sol)

  This defines a public key encryption format, based on ECIES. It uses elliptic curve operations on alt_bn128 G1.
