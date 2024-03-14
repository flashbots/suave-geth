pragma solidity ^0.8.8;

import "../libraries/Suave.sol";

contract AnyBundleContract {
    event DataRecordEvent(Suave.DataId dataId, uint64 decryptionCondition, address[] allowedPeekers);

    function fetchConfidentialBundleData() public returns (bytes memory) {
        require(Suave.isConfidential());

        bytes memory confidentialInputs = Suave.confidentialInputs();
        return abi.decode(confidentialInputs, (bytes));
    }

    function simulateBundle(bytes memory bundleData) public returns (uint64) {
        require(Suave.isConfidential());

        string memory session = Suave.newBuilder();
        return Suave.simulateBundle(session, bundleData).egp;
    }

    function emitDataRecord(Suave.DataRecord calldata dataRecord) public {
        emit DataRecordEvent(dataRecord.id, dataRecord.decryptionCondition, dataRecord.allowedPeekers);
    }
}

contract BundleContract is AnyBundleContract {
    function newBundle(
        uint64 decryptionCondition,
        address[] memory dataAllowedPeekers,
        address[] memory dataAllowedStores
    ) external payable returns (bytes memory) {
        require(Suave.isConfidential());

        bytes memory bundleData = this.fetchConfidentialBundleData();

        uint64 egp = simulateBundle(bundleData);

        Suave.DataRecord memory dataRecord =
            Suave.newDataRecord(decryptionCondition, dataAllowedPeekers, dataAllowedStores, "default:v0:ethBundles");

        Suave.confidentialStore(dataRecord.id, "default:v0:ethBundles", bundleData);
        Suave.confidentialStore(dataRecord.id, "default:v0:ethBundleSimResults", abi.encode(egp));

        return emitAndReturn(dataRecord, bundleData);
    }

    function emitAndReturn(Suave.DataRecord memory dataRecord, bytes memory) internal virtual returns (bytes memory) {
        emit DataRecordEvent(dataRecord.id, dataRecord.decryptionCondition, dataRecord.allowedPeekers);
        return bytes.concat(this.emitDataRecord.selector, abi.encode(dataRecord));
    }
}

contract EthBundleSenderContract is BundleContract {
    string[] public builderUrls;

    constructor(string[] memory builderUrls_) {
        builderUrls = builderUrls_;
    }

    function emitAndReturn(Suave.DataRecord memory dataRecord, bytes memory bundleData)
        internal
        virtual
        override
        returns (bytes memory)
    {
        for (uint256 i = 0; i < builderUrls.length; i++) {
            Suave.submitBundleJsonRPC(builderUrls[i], "eth_sendBundle", bundleData);
        }

        return BundleContract.emitAndReturn(dataRecord, bundleData);
    }
}

contract MevShareContract is AnyBundleContract {
    event HintEvent(Suave.DataId dataId, bytes hint);

    event MatchEvent(Suave.DataId matchDataId, bytes matchHint);

    function newTransaction(
        uint64 decryptionCondition,
        address[] memory dataAllowedPeekers,
        address[] memory dataAllowedStores
    ) external payable returns (bytes memory) {
        // 0. check confidential execution
        require(Suave.isConfidential());

        // 1. fetch bundle data
        bytes memory bundleData = this.fetchConfidentialBundleData();

        // 2. sim bundle
        uint64 egp = simulateBundle(bundleData);

        // 3. extract hint
        bytes memory hint = Suave.extractHint(bundleData);

        // // 4. store bundle and sim results
        Suave.DataRecord memory dataRecord = Suave.newDataRecord(
            decryptionCondition, dataAllowedPeekers, dataAllowedStores, "mevshare:v0:unmatchedBundles"
        );
        Suave.confidentialStore(dataRecord.id, "mevshare:v0:ethBundles", bundleData);
        Suave.confidentialStore(dataRecord.id, "mevshare:v0:ethBundleSimResults", abi.encode(egp));
        emit DataRecordEvent(dataRecord.id, dataRecord.decryptionCondition, dataRecord.allowedPeekers);
        emit HintEvent(dataRecord.id, hint);

        // // 5. return "callback" to emit hint onchain
        return bytes.concat(this.emitDataRecordAndHint.selector, abi.encode(dataRecord, hint));
    }

    function emitDataRecordAndHint(Suave.DataRecord calldata dataRecord, bytes memory hint) public {
        emit DataRecordEvent(dataRecord.id, dataRecord.decryptionCondition, dataRecord.allowedPeekers);
        emit HintEvent(dataRecord.id, hint);
    }

    function newMatch(
        uint64 decryptionCondition,
        address[] memory dataAllowedPeekers,
        address[] memory dataAllowedStores,
        Suave.DataId sharedataId
    ) external payable returns (bytes memory) {
        // WARNING : this function will copy the original mev share bid
        // into a new key with potentially different permsissions

        require(Suave.isConfidential());
        // 1. fetch confidential data
        bytes memory matchBundleData = this.fetchConfidentialBundleData();

        // 2. sim match alone for validity
        uint64 egp = simulateBundle(matchBundleData);

        // 3. extract hint
        bytes memory matchHint = Suave.extractHint(matchBundleData);

        Suave.DataRecord memory dataRecord = Suave.newDataRecord(
            decryptionCondition, dataAllowedPeekers, dataAllowedStores, "mevshare:v0:matchDataRecords"
        );
        Suave.confidentialStore(dataRecord.id, "mevshare:v0:ethBundles", matchBundleData);
        Suave.confidentialStore(dataRecord.id, "mevshare:v0:ethBundleSimResults", abi.encode(egp));

        //4. merge data records
        Suave.DataId[] memory dataRecords = new Suave.DataId[](2);
        dataRecords[0] = sharedataId;
        dataRecords[1] = dataRecord.id;
        Suave.confidentialStore(dataRecord.id, "mevshare:v0:mergedDataRecords", abi.encode(dataRecords));

        return emitMatchDataRecordAndHint(dataRecord, matchHint);
    }

    function emitMatchDataRecordAndHint(Suave.DataRecord memory dataRecord, bytes memory matchHint)
        internal
        virtual
        returns (bytes memory)
    {
        emit DataRecordEvent(dataRecord.id, dataRecord.decryptionCondition, dataRecord.allowedPeekers);
        emit MatchEvent(dataRecord.id, matchHint);

        return bytes.concat(this.emitDataRecord.selector, abi.encode(dataRecord));
    }
}

contract MevShareBundleSenderContract is MevShareContract {
    string[] public builderUrls;

    constructor(string[] memory builderUrls_) {
        builderUrls = builderUrls_;
    }

    function emitMatchDataRecordAndHint(Suave.DataRecord memory dataRecord, bytes memory matchHint)
        internal
        virtual
        override
        returns (bytes memory)
    {
        bytes memory bundleData = Suave.fillMevShareBundle(dataRecord.id);
        for (uint256 i = 0; i < builderUrls.length; i++) {
            Suave.submitBundleJsonRPC(builderUrls[i], "mev_sendBundle", bundleData);
        }

        return MevShareContract.emitMatchDataRecordAndHint(dataRecord, matchHint);
    }
}

/* Not tested or implemented on the precompile side */
struct EgpRecordPair {
    uint64 egp; // in wei, beware overflow
    Suave.DataId dataId;
}

contract EthBlockContract is AnyBundleContract {
    event BuilderBoostBidEvent(Suave.DataId dataId, bytes builderBid);

    function idsEqual(Suave.DataId _l, Suave.DataId _r) public pure returns (bool) {
        bytes memory l = abi.encodePacked(_l);
        bytes memory r = abi.encodePacked(_r);
        for (uint256 i = 0; i < l.length; i++) {
            if (bytes(l)[i] != r[i]) {
                return false;
            }
        }

        return true;
    }

    function buildMevShare(Suave.BuildBlockArgs memory blockArgs, uint64 blockHeight) public returns (bytes memory) {
        require(Suave.isConfidential());

        Suave.DataRecord[] memory allShareMatchDataRecords =
            Suave.fetchDataRecords(blockHeight, "mevshare:v0:matchDataRecords");
        Suave.DataRecord[] memory allShareUserDataRecords =
            Suave.fetchDataRecords(blockHeight, "mevshare:v0:unmatchedBundles");

        if (allShareUserDataRecords.length == 0) {
            revert Suave.PeekerReverted(address(this), "no data records");
        }

        Suave.DataRecord[] memory allRecords = new Suave.DataRecord[](allShareUserDataRecords.length);
        for (uint256 i = 0; i < allShareUserDataRecords.length; i++) {
            // TODO: sort matches by egp first!
            Suave.DataRecord memory dataRecordToInsert = allShareUserDataRecords[i]; // will be updated with the best match if any
            for (uint256 j = 0; j < allShareMatchDataRecords.length; j++) {
                // TODO: should be done once at the start and sorted
                Suave.DataId[] memory mergeddataIds = abi.decode(
                    Suave.confidentialRetrieve(allShareMatchDataRecords[j].id, "mevshare:v0:mergedDataRecords"),
                    (Suave.DataId[])
                );
                if (idsEqual(mergeddataIds[0], allShareUserDataRecords[i].id)) {
                    dataRecordToInsert = allShareMatchDataRecords[j];
                    break;
                }
            }
            allRecords[i] = dataRecordToInsert;
        }

        EgpRecordPair[] memory bidsByEGP = new EgpRecordPair[](allRecords.length);
        for (uint256 i = 0; i < allRecords.length; i++) {
            bytes memory simResults = Suave.confidentialRetrieve(allRecords[i].id, "mevshare:v0:ethBundleSimResults");
            uint64 egp = abi.decode(simResults, (uint64));
            bidsByEGP[i] = EgpRecordPair(egp, allRecords[i].id);
        }

        // Bubble sort, cause why not
        uint256 n = bidsByEGP.length;
        for (uint256 i = 0; i < n - 1; i++) {
            for (uint256 j = i + 1; j < n; j++) {
                if (bidsByEGP[i].egp < bidsByEGP[j].egp) {
                    EgpRecordPair memory temp = bidsByEGP[i];
                    bidsByEGP[i] = bidsByEGP[j];
                    bidsByEGP[j] = temp;
                }
            }
        }

        Suave.DataId[] memory alldataIds = new Suave.DataId[](allRecords.length);
        for (uint256 i = 0; i < bidsByEGP.length; i++) {
            alldataIds[i] = bidsByEGP[i].dataId;
        }

        return buildAndEmit(blockArgs, blockHeight, alldataIds, "mevshare:v0");
    }

    function buildFromPool(Suave.BuildBlockArgs memory blockArgs, uint64 blockHeight) public returns (bytes memory) {
        require(Suave.isConfidential());

        Suave.DataRecord[] memory allRecords = Suave.fetchDataRecords(blockHeight, "default:v0:ethBundles");
        if (allRecords.length == 0) {
            revert Suave.PeekerReverted(address(this), "no data records");
        }

        EgpRecordPair[] memory bidsByEGP = new EgpRecordPair[](allRecords.length);
        for (uint256 i = 0; i < allRecords.length; i++) {
            bytes memory simResults = Suave.confidentialRetrieve(allRecords[i].id, "default:v0:ethBundleSimResults");
            uint64 egp = abi.decode(simResults, (uint64));
            bidsByEGP[i] = EgpRecordPair(egp, allRecords[i].id);
        }

        // Bubble sort, cause why not
        uint256 n = bidsByEGP.length;
        for (uint256 i = 0; i < n - 1; i++) {
            for (uint256 j = i + 1; j < n; j++) {
                if (bidsByEGP[i].egp < bidsByEGP[j].egp) {
                    EgpRecordPair memory temp = bidsByEGP[i];
                    bidsByEGP[i] = bidsByEGP[j];
                    bidsByEGP[j] = temp;
                }
            }
        }

        Suave.DataId[] memory alldataIds = new Suave.DataId[](allRecords.length);
        for (uint256 i = 0; i < bidsByEGP.length; i++) {
            alldataIds[i] = bidsByEGP[i].dataId;
        }

        return buildAndEmit(blockArgs, blockHeight, alldataIds, "");
    }

    function buildAndEmit(
        Suave.BuildBlockArgs memory blockArgs,
        uint64 blockHeight,
        Suave.DataId[] memory records,
        string memory namespace
    ) public virtual returns (bytes memory) {
        require(Suave.isConfidential());

        (Suave.DataRecord memory blockBid, bytes memory builderBid) =
            this.doBuild(blockArgs, blockHeight, records, namespace);

        emit BuilderBoostBidEvent(blockBid.id, builderBid);
        emit DataRecordEvent(blockBid.id, blockBid.decryptionCondition, blockBid.allowedPeekers);
        return bytes.concat(this.emitBuilderBidAndBid.selector, abi.encode(blockBid, builderBid));
    }

    function doBuild(
        Suave.BuildBlockArgs memory blockArgs,
        uint64 blockHeight,
        Suave.DataId[] memory records,
        string memory namespace
    ) public returns (Suave.DataRecord memory, bytes memory) {
        address[] memory allowedPeekers = new address[](2);
        allowedPeekers[0] = address(this);
        allowedPeekers[1] = Suave.BUILD_ETH_BLOCK;

        Suave.DataRecord memory blockBid =
            Suave.newDataRecord(blockHeight, allowedPeekers, allowedPeekers, "default:v0:mergedDataRecords");
        Suave.confidentialStore(blockBid.id, "default:v0:mergedDataRecords", abi.encode(records));

        (bytes memory builderBid, bytes memory payload) = Suave.buildEthBlock(blockArgs, blockBid.id, namespace);
        Suave.confidentialStore(blockBid.id, "default:v0:builderPayload", payload); // only through this.unlock

        return (blockBid, builderBid);
    }

    function emitBuilderBidAndBid(Suave.DataRecord memory dataRecord, bytes memory builderBid)
        public
        returns (Suave.DataRecord memory, bytes memory)
    {
        emit BuilderBoostBidEvent(dataRecord.id, builderBid);
        emit DataRecordEvent(dataRecord.id, dataRecord.decryptionCondition, dataRecord.allowedPeekers);
        return (dataRecord, builderBid);
    }

    function unlock(Suave.DataId dataId, bytes memory signedBlindedHeader) public returns (bytes memory) {
        require(Suave.isConfidential());

        // TODO: verify the header is correct
        // TODO: incorporate protocol name
        bytes memory payload = Suave.confidentialRetrieve(dataId, "default:v0:builderPayload");
        return payload;
    }
}

contract EthBlockBidSenderContract is EthBlockContract {
    string boostRelayUrl;

    constructor(string memory boostRelayUrl_) {
        boostRelayUrl = boostRelayUrl_;
    }

    function buildAndEmit(
        Suave.BuildBlockArgs memory blockArgs,
        uint64 blockHeight,
        Suave.DataId[] memory dataRecords,
        string memory namespace
    ) public virtual override returns (bytes memory) {
        require(Suave.isConfidential());

        (Suave.DataRecord memory blockDataRecord, bytes memory builderBid) =
            this.doBuild(blockArgs, blockHeight, dataRecords, namespace);
        Suave.submitEthBlockToRelay(boostRelayUrl, builderBid);

        emit DataRecordEvent(blockDataRecord.id, blockDataRecord.decryptionCondition, blockDataRecord.allowedPeekers);
        return bytes.concat(this.emitDataRecord.selector, abi.encode(blockDataRecord));
    }
}
