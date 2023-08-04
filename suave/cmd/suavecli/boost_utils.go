package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/r3labs/sse"
)

func SubscribeToPayloadAttributesEvents(ctx context.Context, endpoint string, payloadAttrC chan PayloadAttributesEvent) {
	eventsURL := fmt.Sprintf("%s/eth/v1/events?topics=payload_attributes", endpoint)
	log.Info("subscribing to payload_attributes events")

	for {
		client := sse.NewClient(eventsURL)
		err := client.SubscribeRawWithContext(ctx, func(msg *sse.Event) {
			var payloadAttributesResp PayloadAttributesEvent
			err := json.Unmarshal(msg.Data, &payloadAttributesResp)
			if err != nil {
				log.Error("could not unmarshal payload_attributes event", "err", err)
				return
			}

			payloadAttrC <- payloadAttributesResp
		})
		if err != nil {
			log.Error("failed to subscribe to payload_attributes events", "err", err)
			time.Sleep(1 * time.Second)
		}
		log.Warn("beaconclient SubscribeRaw ended, reconnecting")
	}
}

func getValidatorForSlot(relayUrl string, nextSlot uint64) (ValidatorData, error) {
	endpoint := relayUrl + "/relay/v1/builder/validators"
	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, endpoint, nil)
	if err != nil {
		return ValidatorData{}, fmt.Errorf("could not prepare request: %w", err)
	}

	// Execute request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ValidatorData{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return ValidatorData{}, errors.New("nothing returned from the boost relay")
	}

	if resp.StatusCode > 299 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return ValidatorData{}, fmt.Errorf("could not read error response body for status code %d: %w", resp.StatusCode, err)
		}
		return ValidatorData{}, fmt.Errorf("http error: %d / %s", resp.StatusCode, string(bodyBytes))
	}

	var dst GetValidatorRelayResponse
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return ValidatorData{}, fmt.Errorf("could not read response body: %w", err)
	}

	if err := json.Unmarshal(bodyBytes, &dst); err != nil {
		return ValidatorData{}, fmt.Errorf("could not unmarshal response %s: %w", string(bodyBytes), err)
	}

	res := make(map[uint64]ValidatorData)
	for _, data := range dst {
		feeRecipient := common.HexToAddress(data.Entry.Message.FeeRecipient)
		pubkeyHex := strings.ToLower(data.Entry.Message.Pubkey)

		res[data.Slot] = ValidatorData{
			Pubkey:       pubkeyHex,
			FeeRecipient: feeRecipient,
			GasLimit:     data.Entry.Message.GasLimit,
		}
	}
	v, found := res[nextSlot]
	if !found {
		return ValidatorData{}, errors.New("validator not found")
	}

	return v, nil
}

type PayloadAttributesEvent struct {
	Version string                     `json:"version"`
	Data    PayloadAttributesEventData `json:"data"`
}

type PayloadAttributesEventData struct {
	ProposalSlot      uint64            `json:"proposal_slot,string"`
	ParentBlockHash   common.Hash       `json:"parent_block_hash"`
	PayloadAttributes PayloadAttributes `json:"payload_attributes"`
}

type PayloadAttributes struct {
	Timestamp             uint64                `json:"timestamp,string"`
	PrevRandao            common.Hash           `json:"prev_randao"`
	SuggestedFeeRecipient common.Address        `json:"suggested_fee_recipient"`
	Withdrawals           []*capella.Withdrawal `json:"withdrawals"`
}

type ValidatorData struct {
	Pubkey       string
	FeeRecipient common.Address
	GasLimit     uint64
}

type GetValidatorRelayResponse []struct {
	Slot  uint64 `json:"slot,string"`
	Entry struct {
		Message struct {
			FeeRecipient string `json:"fee_recipient"`
			GasLimit     uint64 `json:"gas_limit,string"`
			Timestamp    uint64 `json:"timestamp,string"`
			Pubkey       string `json:"pubkey"`
		} `json:"message"`
		Signature string `json:"signature"`
	} `json:"entry"`
}
