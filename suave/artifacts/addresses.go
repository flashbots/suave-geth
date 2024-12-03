// Code generated by suave/gen. DO NOT EDIT.
// Hash: 93d0334f831efa37bf623305bc1fadbb86200c71f9e3c4a97337f7879f4216c1
package artifacts

import (
	"github.com/ethereum/go-ethereum/common"
)

// List of suave precompile addresses
var (
	aesDecryptAddr            = common.HexToAddress("0x000000000000000000000000000000005670000d")
	aesEncryptAddr            = common.HexToAddress("0x000000000000000000000000000000005670000e")
	buildEthBlockAddr         = common.HexToAddress("0x0000000000000000000000000000000042100001")
	buildEthBlockToAddr       = common.HexToAddress("0x0000000000000000000000000000000042100006")
	confidentialInputsAddr    = common.HexToAddress("0x0000000000000000000000000000000042010001")
	confidentialRetrieveAddr  = common.HexToAddress("0x0000000000000000000000000000000042020001")
	confidentialStoreAddr     = common.HexToAddress("0x0000000000000000000000000000000042020000")
	contextGetAddr            = common.HexToAddress("0x0000000000000000000000000000000053300003")
	doHTTPRequestAddr         = common.HexToAddress("0x0000000000000000000000000000000043200002")
	doHTTPRequest2Addr        = common.HexToAddress("0x0000000000000000000000000000000043200003")
	ethcallAddr               = common.HexToAddress("0x0000000000000000000000000000000042100003")
	extractHintAddr           = common.HexToAddress("0x0000000000000000000000000000000042100037")
	fetchDataRecordsAddr      = common.HexToAddress("0x0000000000000000000000000000000042030001")
	fillMevShareBundleAddr    = common.HexToAddress("0x0000000000000000000000000000000043200001")
	getInsecureTimeAddr       = common.HexToAddress("0x000000000000000000000000000000007770000c")
	newBuilderAddr            = common.HexToAddress("0x0000000000000000000000000000000053200001")
	newDataRecordAddr         = common.HexToAddress("0x0000000000000000000000000000000042030000")
	privateKeyGenAddr         = common.HexToAddress("0x0000000000000000000000000000000053200003")
	randomBytesAddr           = common.HexToAddress("0x000000000000000000000000000000007770000b")
	signEthTransactionAddr    = common.HexToAddress("0x0000000000000000000000000000000040100001")
	signMessageAddr           = common.HexToAddress("0x0000000000000000000000000000000040100003")
	simulateBundleAddr        = common.HexToAddress("0x0000000000000000000000000000000042100000")
	simulateTransactionAddr   = common.HexToAddress("0x0000000000000000000000000000000053200002")
	submitBundleJsonRPCAddr   = common.HexToAddress("0x0000000000000000000000000000000043000001")
	submitEthBlockToRelayAddr = common.HexToAddress("0x0000000000000000000000000000000042100002")
)

var SuaveMethods = map[string]common.Address{
	"aesDecrypt":            aesDecryptAddr,
	"aesEncrypt":            aesEncryptAddr,
	"buildEthBlock":         buildEthBlockAddr,
	"buildEthBlockTo":       buildEthBlockToAddr,
	"confidentialInputs":    confidentialInputsAddr,
	"confidentialRetrieve":  confidentialRetrieveAddr,
	"confidentialStore":     confidentialStoreAddr,
	"contextGet":            contextGetAddr,
	"doHTTPRequest":         doHTTPRequestAddr,
	"doHTTPRequest2":        doHTTPRequest2Addr,
	"ethcall":               ethcallAddr,
	"extractHint":           extractHintAddr,
	"fetchDataRecords":      fetchDataRecordsAddr,
	"fillMevShareBundle":    fillMevShareBundleAddr,
	"getInsecureTime":       getInsecureTimeAddr,
	"newBuilder":            newBuilderAddr,
	"newDataRecord":         newDataRecordAddr,
	"privateKeyGen":         privateKeyGenAddr,
	"randomBytes":           randomBytesAddr,
	"signEthTransaction":    signEthTransactionAddr,
	"signMessage":           signMessageAddr,
	"simulateBundle":        simulateBundleAddr,
	"simulateTransaction":   simulateTransactionAddr,
	"submitBundleJsonRPC":   submitBundleJsonRPCAddr,
	"submitEthBlockToRelay": submitEthBlockToRelayAddr,
}

func PrecompileAddressToName(addr common.Address) string {
	switch addr {
	case aesDecryptAddr:
		return "aesDecrypt"
	case aesEncryptAddr:
		return "aesEncrypt"
	case buildEthBlockAddr:
		return "buildEthBlock"
	case buildEthBlockToAddr:
		return "buildEthBlockTo"
	case confidentialInputsAddr:
		return "confidentialInputs"
	case confidentialRetrieveAddr:
		return "confidentialRetrieve"
	case confidentialStoreAddr:
		return "confidentialStore"
	case contextGetAddr:
		return "contextGet"
	case doHTTPRequestAddr:
		return "doHTTPRequest"
	case doHTTPRequest2Addr:
		return "doHTTPRequest2"
	case ethcallAddr:
		return "ethcall"
	case extractHintAddr:
		return "extractHint"
	case fetchDataRecordsAddr:
		return "fetchDataRecords"
	case fillMevShareBundleAddr:
		return "fillMevShareBundle"
	case getInsecureTimeAddr:
		return "getInsecureTime"
	case newBuilderAddr:
		return "newBuilder"
	case newDataRecordAddr:
		return "newDataRecord"
	case privateKeyGenAddr:
		return "privateKeyGen"
	case randomBytesAddr:
		return "randomBytes"
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
