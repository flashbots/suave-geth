package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

var commands = map[string]func(){
	"sendBundle":                cmdSendBundle,
	"deployBlockSenderContract": cmdDeployBlockSenderContract,
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
	isOffchainAddress = common.HexToAddress("0x42010000")

	fetchBidsAddress    = common.HexToAddress("0x42030001")
	newBundleBidAddress = common.HexToAddress("0x42200000")
	newBlockBidAddress  = common.HexToAddress("0x42200001")

	// simulateBundleAddress = common.HexToAddress("0x42100000")
	buildEthBlockAddress = common.HexToAddress("0x42100001")
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
		return errors.Join(rpcErr, err)
	}

	unpacked, err := suaveLibAbi.Errors["PeekerReverted"].Inputs.Unpack(decodedError[4:])
	if err != nil {
		return errors.Join(rpcErr, err)
	}

	revertReason := string(unpacked[1].([]byte))
	return errors.Join(rpcErr, fmt.Errorf("revert reason: %s", revertReason))
}
