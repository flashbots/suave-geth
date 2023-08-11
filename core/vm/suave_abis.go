package vm

var bidIdsAbi = mustParseMethodAbi(`[{"inputs": [{ "type": "bytes16[]" }], "name": "bidids", "outputs":[], "type": "function"}]`, "bidids")

var precompilesAbi = mustParseAbi(`[
    {
      "inputs": [
        {
          "internalType": "address",
          "name": "",
          "type": "address"
        },
        {
          "internalType": "bytes",
          "name": "",
          "type": "bytes"
        }
      ],
      "name": "PeekerReverted",
      "type": "error"
    },
    {
      "anonymous": false,
      "inputs": [
        {
          "indexed": false,
          "internalType": "Suave.BidId[]",
          "name": "ids",
          "type": "bytes16[]"
        }
      ],
      "name": "BidIds",
      "type": "event"
    },
    {
      "inputs": [
        {
          "components": [
            {
              "internalType": "uint64",
              "name": "slot",
              "type": "uint64"
            },
            {
              "internalType": "bytes",
              "name": "proposerPubkey",
              "type": "bytes"
            },
            {
              "internalType": "bytes32",
              "name": "parent",
              "type": "bytes32"
            },
            {
              "internalType": "uint64",
              "name": "timestamp",
              "type": "uint64"
            },
            {
              "internalType": "address",
              "name": "feeRecipient",
              "type": "address"
            },
            {
              "internalType": "uint64",
              "name": "gasLimit",
              "type": "uint64"
            },
            {
              "internalType": "bytes32",
              "name": "random",
              "type": "bytes32"
            },
            {
              "components": [
                {
                  "internalType": "uint64",
                  "name": "index",
                  "type": "uint64"
                },
                {
                  "internalType": "uint64",
                  "name": "validator",
                  "type": "uint64"
                },
                {
                  "internalType": "address",
                  "name": "Address",
                  "type": "address"
                },
                {
                  "internalType": "uint64",
                  "name": "amount",
                  "type": "uint64"
                }
              ],
              "internalType": "struct Suave.Withdrawal[]",
              "name": "withdrawals",
              "type": "tuple[]"
            }
          ],
          "internalType": "struct Suave.BuildBlockArgs",
          "name": "blockArgs",
          "type": "tuple"
        },
        {
          "internalType": "Suave.BidId",
          "name": "bid",
          "type": "bytes16"
        },
        {
          "internalType": "string",
          "name": "namespace",
          "type": "string"
        }
      ],
      "name": "buildEthBlock",
      "outputs": [
        {
          "internalType": "bytes",
          "name": "",
          "type": "bytes"
        },
        {
          "internalType": "bytes",
          "name": "",
          "type": "bytes"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "Suave.BidId",
          "name": "bidId",
          "type": "bytes16"
        },
        {
          "internalType": "string",
          "name": "key",
          "type": "string"
        }
      ],
      "name": "confidentialStoreRetrieve",
      "outputs": [
        {
          "internalType": "bytes",
          "name": "",
          "type": "bytes"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "Suave.BidId",
          "name": "bidId",
          "type": "bytes16"
        },
        {
          "internalType": "string",
          "name": "key",
          "type": "string"
        },
        {
          "internalType": "bytes",
          "name": "data",
          "type": "bytes"
        }
      ],
      "name": "confidentialStoreStore",
      "outputs": [],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "bytes",
          "name": "bundleData",
          "type": "bytes"
        }
      ],
      "name": "extractHint",
      "outputs": [
        {
          "internalType": "bytes",
          "name": "",
          "type": "bytes"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "uint64",
          "name": "cond",
          "type": "uint64"
        },
        {
          "internalType": "string",
          "name": "namespace",
          "type": "string"
        }
      ],
      "name": "fetchBids",
      "outputs": [
        {
          "components": [
            {
              "internalType": "Suave.BidId",
              "name": "id",
              "type": "bytes16"
            },
            {
              "internalType": "uint64",
              "name": "decryptionCondition",
              "type": "uint64"
            },
            {
              "internalType": "address[]",
              "name": "allowedPeekers",
              "type": "address[]"
            }
          ],
          "internalType": "struct Suave.Bid[]",
          "name": "",
          "type": "tuple[]"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "uint64",
          "name": "decryptionCondition",
          "type": "uint64"
        },
        {
          "internalType": "address[]",
          "name": "allowedPeekers",
          "type": "address[]"
        },
        {
          "internalType": "string",
          "name": "BidType",
          "type": "string"
        }
      ],
      "name": "newBid",
      "outputs": [
        {
          "components": [
            {
              "internalType": "Suave.BidId",
              "name": "id",
              "type": "bytes16"
            },
            {
              "internalType": "uint64",
              "name": "decryptionCondition",
              "type": "uint64"
            },
            {
              "internalType": "address[]",
              "name": "allowedPeekers",
              "type": "address[]"
            }
          ],
          "internalType": "struct Suave.Bid",
          "name": "",
          "type": "tuple"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "bytes",
          "name": "bundleData",
          "type": "bytes"
        }
      ],
      "name": "simulateBundle",
      "outputs": [
        {
          "internalType": "uint64",
          "name": "",
          "type": "uint64"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "string",
          "name": "relayUrl",
          "type": "string"
        },
        {
          "internalType": "bytes",
          "name": "builderBid",
          "type": "bytes"
        }
      ],
      "name": "submitEthBlockBidToRelay",
      "outputs": [
        {
          "internalType": "bytes",
          "name": "",
          "type": "bytes"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    }
  ]`)
