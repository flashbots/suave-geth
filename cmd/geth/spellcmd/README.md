# Spell command

## Deploy a contract

To deploy a contract, you need to be in the root of a Forge project that contains the built artifacts and run:

```bash
$ suave-geth spell deploy [--artifacts out] <solidity-file>:<contract-name>
```

Example:

```bash
$ suave-geth spell deploy MyContract.sol:MyContract
```

## Send a confidential compute request

```bash
$ suave-geth spell conf-request [--confidential-input <input>] <contract-addr> '<function signature>' ['(arg1,arg2)']
```

Example:

```bash
$ suave-geth spell conf-request 0x1234567890abcdef1234567890abcdef12345678 'set(uint256)' '(42)'
```
