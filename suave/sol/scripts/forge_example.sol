// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import "../libraries/SuaveForge.sol";
import "forge-std/Script.sol";

contract Example is Script {
    address[] public addressList = [0xC8df3686b4Afb2BB53e60EAe97EF043FE03Fb829];

    function run() public {
        Suave.Bid memory bid = SuaveForge.newBid(0, addressList, addressList, "default:v0:ethBundles");

        Suave.Bid[] memory allShareMatchBids = SuaveForge.fetchBids(0, "default:v0:ethBundles");
        console.log(allShareMatchBids.length);

        SuaveForge.confidentialStoreStore(bid.id, "a", abi.encodePacked("bbbbbb"));
        bytes memory result = SuaveForge.confidentialStoreRetrieve(bid.id, "a");
        console.logBytes(result);

        uint256 EthPriceNow = SuaveForge.getBinancePrice("ETHUSDT");
        // uint256 EthPriceNow = 0;
        console.log(EthPriceNow);
    }
}

contract BrokeredDelegation {
    // user register thair ABI, defines access control policy and fees
    // currently assumes there is only Alice and Bob

    struct Account {
        string APIKey;
        string SecretKey;
        address payable staker;
        uint256 fee; // access control + differentiated supply
        uint256 ID;
    }

    Account[] stakedAccounts;
    uint256 globalID;

    function newDelegation() external payable {
        require(Suave.isConfidential());
        bytes memory confidentialInputs = Suave.confidentialInputs();
		bytes memory keys = abi.decode(confidentialInputs, (bytes));

        require (keys.length == 64+64, "Keys not correct"); // API Key is 64 bytes, as is Secret Key

        bytes memory APIKey = new bytes(64);
		bytes memory SecretKey = new bytes(64);
		
		for (uint i = 0; i < 64; i++) {
			APIKey[i] = keys[i];
		}
		
		for (uint j = 64; j < 64+64; j++) {
			SecretKey[j] = keys[j];
		}
		
		string memory APIKeyString = string(APIKey);
		string memory SecretKeyString = string(SecretKey);
        globalID += 1;

        Account memory newStackedAccount = Account({
			APIKey: APIKeyString,
			SecretKey: SecretKeyString,
			staker: msg.sender,
			fee: 100, // charge 1% as default
            ID: globalID
		});
		
		stakedAccounts.push(newStackedAccount);
    }

    // buy ETH using another person's account, automatically get charged 1%
    function useDelegationByID(uint256 id, string memory ticker) external payable {
        Account memory chosen = stakedAccounts[id];
        
        chosen.staker.transfer(msg.value); // in 1e18 wei
        require(msg.value != 0, "didn't tribute fee to staker");
        SuaveForge.forwardBinanceBuy(chosen.APIKey, chosen.SecretKey, ticker, msg.value * chosen.fee / 10000);
    }
}