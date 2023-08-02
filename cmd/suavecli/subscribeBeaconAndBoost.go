package main

import (
	"context"
	"flag"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

func cmdSubscribeBeaconAndBoost() {
	flagset := flag.NewFlagSet("cmdSubscribeBeaconAndBoost", flag.ExitOnError)

	var (
		// goerliRpc               = flagset.String("goerli_rpc", "http://127.0.0.1:8545", "address of goerli rpc")
		goerliBeaconRpc = flagset.String("goerli_beacon_rpc", "http://127.0.0.1:5052", "address of goerli beacon rpc")
		boostRelayUrl   = flagset.String("relay_url", "http://127.0.0.1:8091", "address of boost relay that the contract will send blocks to")
		verbosity       = flagset.Int("verbosity", int(log.LvlInfo), "log verbosity (0-5)")
	)

	flagset.Parse(os.Args[2:])

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	log.Root().SetHandler(glogger)
	glogger.Verbosity(log.Lvl(*verbosity))

	payloadAttrC := make(chan PayloadAttributesEvent)
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	go SubscribeToPayloadAttributesEvents(ctx, *goerliBeaconRpc, payloadAttrC)

	for paEvent := range payloadAttrC {
		validatorData, err := getValidatorForSlot(*boostRelayUrl, paEvent.Data.ProposalSlot)
		if err != nil {
			log.Error("could not get validator", "slot", paEvent.Data.ProposalSlot, "err", err)
			continue
		}

		log.Info("got validator", "vd", validatorData)

		payloadArgsTuple := struct {
			Slot           uint64
			ProposerPubkey []byte
			Parent         common.Hash
			Timestamp      uint64
			FeeRecipient   common.Address
			GasLimit       uint64
			Random         common.Hash
			Withdrawals    []struct {
				Index     uint64
				Validator uint64
				Address   common.Address
				Amount    uint64
			}
		}{
			Slot:           paEvent.Data.ProposalSlot,
			Parent:         paEvent.Data.ParentBlockHash,
			Timestamp:      paEvent.Data.PayloadAttributes.Timestamp,
			Random:         paEvent.Data.PayloadAttributes.PrevRandao,
			ProposerPubkey: hexutil.MustDecode(validatorData.Pubkey),
			FeeRecipient:   validatorData.FeeRecipient,
			GasLimit:       validatorData.GasLimit,
		}

		for _, w := range paEvent.Data.PayloadAttributes.Withdrawals {
			payloadArgsTuple.Withdrawals = append(payloadArgsTuple.Withdrawals, struct {
				Index     uint64
				Validator uint64
				Address   common.Address
				Amount    uint64
			}{
				Index:     uint64(w.Index),
				Validator: uint64(w.ValidatorIndex),
				Address:   common.Address(w.Address),
				Amount:    uint64(w.Amount),
			})
		}

		// TODO: build a block using the above! also needs to get goerli block height I guess, but thats simple
		log.Info("PA", "data", payloadArgsTuple)
	}
}
