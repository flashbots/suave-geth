// Hash: c47d1edbb0c5bf09f52a6db80269fdd9cdd62ad7b06b95a4625de00d74ac316f
package artifacts

import (
	"github.com/ethereum/go-ethereum/common"
)

var SuaveMethods = map[string]common.Address{
	"buildEthBlock":             common.HexToAddress("0x0000000000000000000000000000000042100001"),
	"confidentialInputs":        common.HexToAddress("0x0000000000000000000000000000000042010001"),
	"confidentialStoreRetrieve": common.HexToAddress("0x0000000000000000000000000000000042020001"),
	"confidentialStoreStore":    common.HexToAddress("0x0000000000000000000000000000000042020000"),
	"extractHint":               common.HexToAddress("0x0000000000000000000000000000000042100037"),
	"fetchBids":                 common.HexToAddress("0x0000000000000000000000000000000042030001"),
	"newBid":                    common.HexToAddress("0x0000000000000000000000000000000042030000"),
	"simulateBundle":            common.HexToAddress("0x0000000000000000000000000000000042100000"),
	"submitEthBlockBidToRelay":  common.HexToAddress("0x0000000000000000000000000000000042100002"),
}
