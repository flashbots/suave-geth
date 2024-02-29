// Code generated by suave/gen. DO NOT EDIT.
// Hash: 294bd11c203e11658343c4d6b0dbb41cfd95cbce67e2eacf3d18fe165d929e68
package vm

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/suave/artifacts"
	"github.com/mitchellh/mapstructure"
)

var (
	errFailedToUnpackInput = fmt.Errorf("failed to decode input")
	errFailedToDecodeField = fmt.Errorf("failed to decode field")
	errFailedToPackOutput  = fmt.Errorf("failed to encode output")
)

type SuaveRuntime interface {
	buildEthBlock(blockArgs types.BuildBlockArgs, dataId types.DataId, namespace string) ([]byte, []byte, error)
	confidentialInputs() ([]byte, error)
	confidentialRetrieve(dataId types.DataId, key string) ([]byte, error)
	confidentialStore(dataId types.DataId, key string, value []byte) error
	contextGet(key string) ([]byte, error)
	doHTTPRequest(request types.HttpRequest) ([]byte, error)
	ethcall(contractAddr common.Address, input1 []byte) ([]byte, error)
	extractHint(bundleData []byte) ([]byte, error)
	fetchDataRecords(cond uint64, namespace string) ([]types.DataRecord, error)
	fillMevShareBundle(dataId types.DataId) ([]byte, error)
	newBuilder() (string, error)
	newDataRecord(decryptionCondition uint64, allowedPeekers []common.Address, allowedStores []common.Address, dataType string) (types.DataRecord, error)
	privateKeyGen(crypto types.CryptoSignature) (string, error)
	randomBytes(numBytes uint8) ([]byte, error)
	signEthTransaction(txn []byte, chainId string, signingKey string) ([]byte, error)
	signMessage(digest []byte, crypto types.CryptoSignature, signingKey string) ([]byte, error)
	simulateBundle(bundleData []byte) (uint64, error)
	simulateTransaction(sessionid string, txn []byte) (types.SimulateTransactionResult, error)
	submitBundleJsonRPC(url string, method string, params []byte) ([]byte, error)
	submitEthBlockToRelay(relayUrl string, builderBid []byte) ([]byte, error)
}

var (
	buildEthBlockAddr         = common.HexToAddress("0x0000000000000000000000000000000042100001")
	confidentialInputsAddr    = common.HexToAddress("0x0000000000000000000000000000000042010001")
	confidentialRetrieveAddr  = common.HexToAddress("0x0000000000000000000000000000000042020001")
	confidentialStoreAddr     = common.HexToAddress("0x0000000000000000000000000000000042020000")
	contextGetAddr            = common.HexToAddress("0x0000000000000000000000000000000053300003")
	doHTTPRequestAddr         = common.HexToAddress("0x0000000000000000000000000000000043200002")
	ethcallAddr               = common.HexToAddress("0x0000000000000000000000000000000042100003")
	extractHintAddr           = common.HexToAddress("0x0000000000000000000000000000000042100037")
	fetchDataRecordsAddr      = common.HexToAddress("0x0000000000000000000000000000000042030001")
	fillMevShareBundleAddr    = common.HexToAddress("0x0000000000000000000000000000000043200001")
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

var addrList = []common.Address{
	buildEthBlockAddr, confidentialInputsAddr, confidentialRetrieveAddr, confidentialStoreAddr, contextGetAddr, doHTTPRequestAddr, ethcallAddr, extractHintAddr, fetchDataRecordsAddr, fillMevShareBundleAddr, newBuilderAddr, newDataRecordAddr, privateKeyGenAddr, randomBytesAddr, signEthTransactionAddr, signMessageAddr, simulateBundleAddr, simulateTransactionAddr, submitBundleJsonRPCAddr, submitEthBlockToRelayAddr,
}

type SuaveRuntimeAdapter struct {
	impl SuaveRuntime
}

func (b *SuaveRuntimeAdapter) run(addr common.Address, input []byte) ([]byte, error) {
	switch addr {
	case buildEthBlockAddr:
		return b.buildEthBlock(input)

	case confidentialInputsAddr:
		return b.confidentialInputs(input)

	case confidentialRetrieveAddr:
		return b.confidentialRetrieve(input)

	case confidentialStoreAddr:
		return b.confidentialStore(input)

	case contextGetAddr:
		return b.contextGet(input)

	case doHTTPRequestAddr:
		return b.doHTTPRequest(input)

	case ethcallAddr:
		return b.ethcall(input)

	case extractHintAddr:
		return b.extractHint(input)

	case fetchDataRecordsAddr:
		return b.fetchDataRecords(input)

	case fillMevShareBundleAddr:
		return b.fillMevShareBundle(input)

	case newBuilderAddr:
		return b.newBuilder(input)

	case newDataRecordAddr:
		return b.newDataRecord(input)

	case privateKeyGenAddr:
		return b.privateKeyGen(input)

	case randomBytesAddr:
		return b.randomBytes(input)

	case signEthTransactionAddr:
		return b.signEthTransaction(input)

	case signMessageAddr:
		return b.signMessage(input)

	case simulateBundleAddr:
		return b.simulateBundle(input)

	case simulateTransactionAddr:
		return b.simulateTransaction(input)

	case submitBundleJsonRPCAddr:
		return b.submitBundleJsonRPC(input)

	case submitEthBlockToRelayAddr:
		return b.submitEthBlockToRelay(input)

	default:
		return nil, fmt.Errorf("suave precompile not found for " + addr.String())
	}
}

func (b *SuaveRuntimeAdapter) buildEthBlock(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["buildEthBlock"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		blockArgs types.BuildBlockArgs
		dataId    types.DataId
		namespace string
	)

	if err = mapstructure.Decode(unpacked[0], &blockArgs); err != nil {
		err = errFailedToDecodeField
		return
	}

	if err = mapstructure.Decode(unpacked[1], &dataId); err != nil {
		err = errFailedToDecodeField
		return
	}

	namespace = unpacked[2].(string)

	var (
		blockBid         []byte
		executionPayload []byte
	)

	if blockBid, executionPayload, err = b.impl.buildEthBlock(blockArgs, dataId, namespace); err != nil {
		return
	}

	result, err = artifacts.SuaveAbi.Methods["buildEthBlock"].Outputs.Pack(blockBid, executionPayload)
	if err != nil {
		err = errFailedToPackOutput
		return
	}
	return result, nil

}

func (b *SuaveRuntimeAdapter) confidentialInputs(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["confidentialInputs"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var ()

	var (
		confindentialData []byte
	)

	if confindentialData, err = b.impl.confidentialInputs(); err != nil {
		return
	}

	result = confindentialData
	return result, nil

}

func (b *SuaveRuntimeAdapter) confidentialRetrieve(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["confidentialRetrieve"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		dataId types.DataId
		key    string
	)

	if err = mapstructure.Decode(unpacked[0], &dataId); err != nil {
		err = errFailedToDecodeField
		return
	}

	key = unpacked[1].(string)

	var (
		value []byte
	)

	if value, err = b.impl.confidentialRetrieve(dataId, key); err != nil {
		return
	}

	result = value
	return result, nil

}

func (b *SuaveRuntimeAdapter) confidentialStore(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["confidentialStore"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		dataId types.DataId
		key    string
		value  []byte
	)

	if err = mapstructure.Decode(unpacked[0], &dataId); err != nil {
		err = errFailedToDecodeField
		return
	}

	key = unpacked[1].(string)
	value = unpacked[2].([]byte)

	var ()

	if err = b.impl.confidentialStore(dataId, key, value); err != nil {
		return
	}

	return nil, nil

}

func (b *SuaveRuntimeAdapter) contextGet(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["contextGet"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		key string
	)

	key = unpacked[0].(string)

	var (
		value []byte
	)

	if value, err = b.impl.contextGet(key); err != nil {
		return
	}

	result, err = artifacts.SuaveAbi.Methods["contextGet"].Outputs.Pack(value)
	if err != nil {
		err = errFailedToPackOutput
		return
	}
	return result, nil

}

func (b *SuaveRuntimeAdapter) doHTTPRequest(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["doHTTPRequest"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		request types.HttpRequest
	)

	if err = mapstructure.Decode(unpacked[0], &request); err != nil {
		err = errFailedToDecodeField
		return
	}

	var (
		httpResponse []byte
	)

	if httpResponse, err = b.impl.doHTTPRequest(request); err != nil {
		return
	}

	result, err = artifacts.SuaveAbi.Methods["doHTTPRequest"].Outputs.Pack(httpResponse)
	if err != nil {
		err = errFailedToPackOutput
		return
	}
	return result, nil

}

func (b *SuaveRuntimeAdapter) ethcall(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["ethcall"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		contractAddr common.Address
		input1       []byte
	)

	contractAddr = unpacked[0].(common.Address)
	input1 = unpacked[1].([]byte)

	var (
		callOutput []byte
	)

	if callOutput, err = b.impl.ethcall(contractAddr, input1); err != nil {
		return
	}

	result, err = artifacts.SuaveAbi.Methods["ethcall"].Outputs.Pack(callOutput)
	if err != nil {
		err = errFailedToPackOutput
		return
	}
	return result, nil

}

func (b *SuaveRuntimeAdapter) extractHint(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["extractHint"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		bundleData []byte
	)

	bundleData = unpacked[0].([]byte)

	var (
		hints []byte
	)

	if hints, err = b.impl.extractHint(bundleData); err != nil {
		return
	}

	result = hints
	return result, nil

}

func (b *SuaveRuntimeAdapter) fetchDataRecords(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["fetchDataRecords"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		cond      uint64
		namespace string
	)

	cond = unpacked[0].(uint64)
	namespace = unpacked[1].(string)

	var (
		dataRecords []types.DataRecord
	)

	if dataRecords, err = b.impl.fetchDataRecords(cond, namespace); err != nil {
		return
	}

	result, err = artifacts.SuaveAbi.Methods["fetchDataRecords"].Outputs.Pack(dataRecords)
	if err != nil {
		err = errFailedToPackOutput
		return
	}
	return result, nil

}

func (b *SuaveRuntimeAdapter) fillMevShareBundle(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["fillMevShareBundle"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		dataId types.DataId
	)

	if err = mapstructure.Decode(unpacked[0], &dataId); err != nil {
		err = errFailedToDecodeField
		return
	}

	var (
		encodedBundle []byte
	)

	if encodedBundle, err = b.impl.fillMevShareBundle(dataId); err != nil {
		return
	}

	result = encodedBundle
	return result, nil

}

func (b *SuaveRuntimeAdapter) newBuilder(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["newBuilder"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var ()

	var (
		sessionid string
	)

	if sessionid, err = b.impl.newBuilder(); err != nil {
		return
	}

	result, err = artifacts.SuaveAbi.Methods["newBuilder"].Outputs.Pack(sessionid)
	if err != nil {
		err = errFailedToPackOutput
		return
	}
	return result, nil

}

func (b *SuaveRuntimeAdapter) newDataRecord(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["newDataRecord"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		decryptionCondition uint64
		allowedPeekers      []common.Address
		allowedStores       []common.Address
		dataType            string
	)

	decryptionCondition = unpacked[0].(uint64)
	allowedPeekers = unpacked[1].([]common.Address)
	allowedStores = unpacked[2].([]common.Address)
	dataType = unpacked[3].(string)

	var (
		dataRecord types.DataRecord
	)

	if dataRecord, err = b.impl.newDataRecord(decryptionCondition, allowedPeekers, allowedStores, dataType); err != nil {
		return
	}

	result, err = artifacts.SuaveAbi.Methods["newDataRecord"].Outputs.Pack(dataRecord)
	if err != nil {
		err = errFailedToPackOutput
		return
	}
	return result, nil

}

func (b *SuaveRuntimeAdapter) privateKeyGen(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["privateKeyGen"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		crypto types.CryptoSignature
	)

	if err = mapstructure.Decode(unpacked[0], &crypto); err != nil {
		err = errFailedToDecodeField
		return
	}

	var (
		privateKey string
	)

	if privateKey, err = b.impl.privateKeyGen(crypto); err != nil {
		return
	}

	result, err = artifacts.SuaveAbi.Methods["privateKeyGen"].Outputs.Pack(privateKey)
	if err != nil {
		err = errFailedToPackOutput
		return
	}
	return result, nil

}

func (b *SuaveRuntimeAdapter) randomBytes(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["randomBytes"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		numBytes uint8
	)

	numBytes = unpacked[0].(uint8)

	var (
		value []byte
	)

	if value, err = b.impl.randomBytes(numBytes); err != nil {
		return
	}

	result, err = artifacts.SuaveAbi.Methods["randomBytes"].Outputs.Pack(value)
	if err != nil {
		err = errFailedToPackOutput
		return
	}
	return result, nil

}

func (b *SuaveRuntimeAdapter) signEthTransaction(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["signEthTransaction"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		txn        []byte
		chainId    string
		signingKey string
	)

	txn = unpacked[0].([]byte)
	chainId = unpacked[1].(string)
	signingKey = unpacked[2].(string)

	var (
		signedTxn []byte
	)

	if signedTxn, err = b.impl.signEthTransaction(txn, chainId, signingKey); err != nil {
		return
	}

	result, err = artifacts.SuaveAbi.Methods["signEthTransaction"].Outputs.Pack(signedTxn)
	if err != nil {
		err = errFailedToPackOutput
		return
	}
	return result, nil

}

func (b *SuaveRuntimeAdapter) signMessage(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["signMessage"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		digest     []byte
		crypto     types.CryptoSignature
		signingKey string
	)

	digest = unpacked[0].([]byte)

	if err = mapstructure.Decode(unpacked[1], &crypto); err != nil {
		err = errFailedToDecodeField
		return
	}

	signingKey = unpacked[2].(string)

	var (
		signature []byte
	)

	if signature, err = b.impl.signMessage(digest, crypto, signingKey); err != nil {
		return
	}

	result, err = artifacts.SuaveAbi.Methods["signMessage"].Outputs.Pack(signature)
	if err != nil {
		err = errFailedToPackOutput
		return
	}
	return result, nil

}

func (b *SuaveRuntimeAdapter) simulateBundle(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["simulateBundle"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		bundleData []byte
	)

	bundleData = unpacked[0].([]byte)

	var (
		effectiveGasPrice uint64
	)

	if effectiveGasPrice, err = b.impl.simulateBundle(bundleData); err != nil {
		return
	}

	result, err = artifacts.SuaveAbi.Methods["simulateBundle"].Outputs.Pack(effectiveGasPrice)
	if err != nil {
		err = errFailedToPackOutput
		return
	}
	return result, nil

}

func (b *SuaveRuntimeAdapter) simulateTransaction(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["simulateTransaction"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		sessionid string
		txn       []byte
	)

	sessionid = unpacked[0].(string)
	txn = unpacked[1].([]byte)

	var (
		simulationResult types.SimulateTransactionResult
	)

	if simulationResult, err = b.impl.simulateTransaction(sessionid, txn); err != nil {
		return
	}

	result, err = artifacts.SuaveAbi.Methods["simulateTransaction"].Outputs.Pack(simulationResult)
	if err != nil {
		err = errFailedToPackOutput
		return
	}
	return result, nil

}

func (b *SuaveRuntimeAdapter) submitBundleJsonRPC(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["submitBundleJsonRPC"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		url    string
		method string
		params []byte
	)

	url = unpacked[0].(string)
	method = unpacked[1].(string)
	params = unpacked[2].([]byte)

	var (
		errorMessage []byte
	)

	if errorMessage, err = b.impl.submitBundleJsonRPC(url, method, params); err != nil {
		return
	}

	result = errorMessage
	return result, nil

}

func (b *SuaveRuntimeAdapter) submitEthBlockToRelay(input []byte) (res []byte, err error) {
	var (
		unpacked []interface{}
		result   []byte
	)

	_ = unpacked
	_ = result

	unpacked, err = artifacts.SuaveAbi.Methods["submitEthBlockToRelay"].Inputs.Unpack(input)
	if err != nil {
		err = errFailedToUnpackInput
		return
	}

	var (
		relayUrl   string
		builderBid []byte
	)

	relayUrl = unpacked[0].(string)
	builderBid = unpacked[1].([]byte)

	var (
		blockBid []byte
	)

	if blockBid, err = b.impl.submitEthBlockToRelay(relayUrl, builderBid); err != nil {
		return
	}

	result = blockBid
	return result, nil

}
