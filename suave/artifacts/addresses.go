// Code generated by suave/gen. DO NOT EDIT.
// Hash: 910f9d9253e37485f1277ae1903a725f4ca97505e266bc5e0039c120da857f42
package artifacts

import (
	"github.com/ethereum/go-ethereum/common"
)

// List of suave precompile addresses
var (
	buildEthBlockAddr         = common.HexToAddress("0x0000000000000000000000000000000042100001")
	confidentialInputsAddr    = common.HexToAddress("0x0000000000000000000000000000000042010001")
	confidentialRetrieveAddr  = common.HexToAddress("0x0000000000000000000000000000000042020001")
	confidentialStoreAddr     = common.HexToAddress("0x0000000000000000000000000000000042020000")
	doHTTPRequestAddr         = common.HexToAddress("0x0000000000000000000000000000000043200002")
	ethcallAddr               = common.HexToAddress("0x0000000000000000000000000000000042100003")
	extractHintAddr           = common.HexToAddress("0x0000000000000000000000000000000042100037")
	fetchDataRecordsAddr      = common.HexToAddress("0x0000000000000000000000000000000042030001")
	fillMevShareBundleAddr    = common.HexToAddress("0x0000000000000000000000000000000043200001")
	newBuilderAddr            = common.HexToAddress("0x0000000000000000000000000000000053200001")
	newDataRecordAddr         = common.HexToAddress("0x0000000000000000000000000000000042030000")
	privateKeyGenAddr         = common.HexToAddress("0x0000000000000000000000000000000053200003")
	signEthTransactionAddr    = common.HexToAddress("0x0000000000000000000000000000000040100001")
	signMessageAddr           = common.HexToAddress("0x0000000000000000000000000000000040100003")
	simulateBundleAddr        = common.HexToAddress("0x0000000000000000000000000000000042100000")
	simulateTransactionAddr   = common.HexToAddress("0x0000000000000000000000000000000053200002")
	submitBundleJsonRPCAddr   = common.HexToAddress("0x0000000000000000000000000000000043000001")
	submitEthBlockToRelayAddr = common.HexToAddress("0x0000000000000000000000000000000042100002")
)

var SuaveMethods = map[string]common.Address{
	"buildEthBlock":         buildEthBlockAddr,
	"confidentialInputs":    confidentialInputsAddr,
	"confidentialRetrieve":  confidentialRetrieveAddr,
	"confidentialStore":     confidentialStoreAddr,
	"doHTTPRequest":         doHTTPRequestAddr,
	"ethcall":               ethcallAddr,
	"extractHint":           extractHintAddr,
	"fetchDataRecords":      fetchDataRecordsAddr,
	"fillMevShareBundle":    fillMevShareBundleAddr,
	"newBuilder":            newBuilderAddr,
	"newDataRecord":         newDataRecordAddr,
	"privateKeyGen":         privateKeyGenAddr,
	"signEthTransaction":    signEthTransactionAddr,
	"signMessage":           signMessageAddr,
	"simulateBundle":        simulateBundleAddr,
	"simulateTransaction":   simulateTransactionAddr,
	"submitBundleJsonRPC":   submitBundleJsonRPCAddr,
	"submitEthBlockToRelay": submitEthBlockToRelayAddr,
}

func PrecompileAddressToName(addr common.Address) string {
	switch addr {
	case buildEthBlockAddr:
		return "buildEthBlock"
	case confidentialInputsAddr:
		return "confidentialInputs"
	case confidentialRetrieveAddr:
		return "confidentialRetrieve"
	case confidentialStoreAddr:
		return "confidentialStore"
	case doHTTPRequestAddr:
		return "doHTTPRequest"
	case ethcallAddr:
		return "ethcall"
	case extractHintAddr:
		return "extractHint"
	case fetchDataRecordsAddr:
		return "fetchDataRecords"
	case fillMevShareBundleAddr:
		return "fillMevShareBundle"
	case newBuilderAddr:
		return "newBuilder"
	case newDataRecordAddr:
		return "newDataRecord"
	case privateKeyGenAddr:
		return "privateKeyGen"
	case signEthTransactionAddr:
		return "signEthTransaction"
	case signMessageAddr:
		return "signMessage"
	case simulateBundleAddr:
		return "simulateBundle"
	case simulateTransactionAddr:
		return "simulateTransaction"
	case submitBundleJsonRPCAddr:
		return "submitBundleJsonRPC"
	case submitEthBlockToRelayAddr:
		return "submitEthBlockToRelay"
	}
	return ""
}
