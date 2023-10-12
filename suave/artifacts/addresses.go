// Code generated by suave/gen. DO NOT EDIT.
// Hash: 8d09b7eac37abc2de5a3d5cfaf8ba25dc21a7df704db353db08348b91e8724fd
package artifacts

import (
	"github.com/ethereum/go-ethereum/common"
)

// List of suave precompile addresses
var (
	buildEthBlockAddr             = common.HexToAddress("0x0000000000000000000000000000000042100001")
	confidentialInputsAddr        = common.HexToAddress("0x0000000000000000000000000000000042010001")
	confidentialStoreRetrieveAddr = common.HexToAddress("0x0000000000000000000000000000000042020001")
	confidentialStoreStoreAddr    = common.HexToAddress("0x0000000000000000000000000000000042020000")
	ethcallAddr                   = common.HexToAddress("0x0000000000000000000000000000000042100003")
	extractHintAddr               = common.HexToAddress("0x0000000000000000000000000000000042100037")
	fetchBidsAddr                 = common.HexToAddress("0x0000000000000000000000000000000042030001")
	newBidAddr                    = common.HexToAddress("0x0000000000000000000000000000000042030000")
	simulateBundleAddr            = common.HexToAddress("0x0000000000000000000000000000000042100000")
	submitEthBlockBidToRelayAddr  = common.HexToAddress("0x0000000000000000000000000000000042100002")
)

var SuaveMethods = map[string]common.Address{
	"buildEthBlock":             buildEthBlockAddr,
	"confidentialInputs":        confidentialInputsAddr,
	"confidentialStoreRetrieve": confidentialStoreRetrieveAddr,
	"confidentialStoreStore":    confidentialStoreStoreAddr,
	"ethcall":                   ethcallAddr,
	"extractHint":               extractHintAddr,
	"fetchBids":                 fetchBidsAddr,
	"newBid":                    newBidAddr,
	"simulateBundle":            simulateBundleAddr,
	"submitEthBlockBidToRelay":  submitEthBlockBidToRelayAddr,
}

func PrecompileAddressToName(addr common.Address) string {
	switch addr {
	case buildEthBlockAddr:
		return "buildEthBlock"
	case confidentialInputsAddr:
		return "confidentialInputs"
	case confidentialStoreRetrieveAddr:
		return "confidentialStoreRetrieve"
	case confidentialStoreStoreAddr:
		return "confidentialStoreStore"
	case ethcallAddr:
		return "ethcall"
	case extractHintAddr:
		return "extractHint"
	case fetchBidsAddr:
		return "fetchBids"
	case newBidAddr:
		return "newBid"
	case simulateBundleAddr:
		return "simulateBundle"
	case submitEthBlockBidToRelayAddr:
		return "submitEthBlockBidToRelay"
	}
	return ""
}