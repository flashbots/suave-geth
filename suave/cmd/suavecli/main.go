package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	suave "github.com/ethereum/go-ethereum/suave/core"
)

var commands = map[string]func(){
	// deploy
	"deployBlockSenderContract": cmdDeployBlockSenderContract,
	"deployMevShareContract":    cmdDeployMevShareContract,
	// send
	"sendBundle":          cmdSendBundle,
	"sendBundleToBuilder": cmdSendBundleToBuilder,
	"sendMevShareBundle":  cmdSendMevShareBundle,
	"sendMevShareMatch":   cmdSendMevShareMatch,
	"sendBuildShareBlock": cmdSendBuildShareBlock,
	// listeners
	"startHintListener":       cmdHintListener,
	"subscribeBeaconAndBoost": cmdSubscribeBeaconAndBoost,
	"startRelayListener":      cmdRelayListener,

	// e2e test
	"testDeployAndShare": cmdTestDeployAndShare,
	"buildGoerliBlocks":  cmdBuildGoerliBlocks,
}

func getAllowedCommands() string {
	allowedCommands := []string{}
	for cmd := range commands {
		allowedCommands = append(allowedCommands, cmd)
	}
	return strings.Join(allowedCommands, ", ")
}

func main() {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	log.Root().SetHandler(glogger)

	// First, get the command to run
	if len(os.Args) < 2 {
		utils.Fatalf("please specify the command. Possible commands: %s", getAllowedCommands())
		return
	}

	cmd, found := commands[os.Args[1]]
	if !found {
		utils.Fatalf("invalid command %s. please specify the command. Possible commands: %s", os.Args[1], getAllowedCommands())
		return
	}

	cmd()
}

var (
	isConfidentialAddress = common.HexToAddress("0x42010000")

	fetchBidsAddress    = common.HexToAddress("0x42030001")
	newBundleBidAddress = common.HexToAddress("0x42200000")
	newBlockBidAddress  = common.HexToAddress("0x42200001")

	buildEthBlockAddress = common.HexToAddress("0x42100001")
	extractHintAddress   = common.HexToAddress("0x42100037")
)

func RequireNoError(err error) {
	if err != nil {
		utils.Fatalf("%v", err)
	}
}

func RequireNoErrorf(err error, format string, args ...interface{}) {
	if err != nil {
		utils.Fatalf(format, args...)
	}
}

func unwrapPeekerError(rpcErr error) error {
	if rpcErr == nil {
		return nil
	}

	if len(rpcErr.Error()) < 26 {
		return rpcErr
	}
	decodedError, err := hexutil.Decode(rpcErr.Error()[20:])
	if err != nil {
		return fmt.Errorf("%s: %s", rpcErr, err)
	}

	unpacked, err := suaveLibAbi.Errors["PeekerReverted"].Inputs.Unpack(decodedError[4:])
	if err != nil {
		return fmt.Errorf("%s: %s", rpcErr, err)
	}

	revertReason := string(unpacked[1].([]byte))
	return fmt.Errorf("%s: %s", rpcErr, fmt.Errorf("revert reason: %s", revertReason))
}

func waitForTransactionToBeConfirmed(suaveClient *rpc.Client, txHash *common.Hash) {
	for i := 0; i < 10; i++ {
		time.Sleep(time.Duration(1+i/2) * time.Second)

		var r *types.Receipt
		err := suaveClient.Call(&r, "eth_getTransactionReceipt", txHash)
		if err == nil && r != nil {
			log.Info("All is good!", "receipt", r, "block_num", r.BlockNumber)
			return
		}
	}
	utils.Fatalf("did not see the receipt succeed in time. hash: %s", txHash.String())
}

func extractBidId(suaveClient *rpc.Client, txHash common.Hash) (suave.BidId, error) {
	var r *types.Receipt
	err := suaveClient.Call(&r, "eth_getTransactionReceipt", &txHash)
	if err == nil && r != nil {
		unpacked, err := mevShareABI.Events["HintEvent"].Inputs.Unpack(r.Logs[1].Data) // index = 1 because second hint is bid event
		if err != nil {
			return suave.BidId{0}, err
		}
		shareBidId := unpacked[0].([16]byte)
		return shareBidId, nil
	}

	return suave.BidId{0}, err
}

func setUpSuaveAndGoerli(privKeyHex *string, kettleAddressHex *string, suaveRpc *string, goerliRpc *string) (*ecdsa.PrivateKey, common.Address, *rpc.Client, *rpc.Client, types.Signer, types.Signer) {
	privKey, err := crypto.HexToECDSA(*privKeyHex)
	RequireNoErrorf(err, "-nodekeyhex: %v", err)
	/* shush linter */ privKey.Public()

	kettleAddress := common.HexToAddress(*kettleAddressHex)

	suaveClient, err := rpc.DialContext(context.TODO(), *suaveRpc)
	RequireNoErrorf(err, "could not connect to suave rpc: %v", err)

	goerliClient, err := rpc.DialContext(context.TODO(), *goerliRpc)
	RequireNoErrorf(err, "could not connect to goerli rpc: %v", err)

	genesis := core.DefaultSuaveGenesisBlock()
	suaveSigner := types.NewSuaveSigner(genesis.Config.ChainID)

	goerliSigner := types.LatestSigner(core.DefaultGoerliGenesisBlock().Config)

	return privKey, kettleAddress, suaveClient, goerliClient, suaveSigner, goerliSigner
}
