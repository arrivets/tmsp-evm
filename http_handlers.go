package tmsp-evm

import (
    "math/big"
    "net/http"
    "encoding/json"
    "log"

    "github.com/ethereum/go-ethereum/rpc"
    "github.com/ethereum/go-ethereum/rlp"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/accounts"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/core/vm"
)

func accountsHandler(w http.ResponseWriter, r *http.Request, m *Service) {
    state, err := m.getState()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    var al JsonAccountList
    
    accs := m.accountManager.Accounts()
    for _, account := range accs {      
        balance := state.GetBalance(account.Address)
        al.Accounts = append(al.Accounts, JsonAccount{Address: account.Address.Hex(),
            Balance: balance})
    }
    
    js, err := json.Marshal(al)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write(js)
}

func transactionHandler(w http.ResponseWriter, r *http.Request, m *Service) {
    state, err := m.getState()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    decoder := json.NewDecoder(r.Body)
    var txArgs SendTxArgs   
    err = decoder.Decode(&txArgs)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer r.Body.Close()

    tx, err := prepareTransaction(txArgs, state, m.accountManager)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    data, err := rlp.EncodeToBytes(tx)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    err = m.platform.CreateTransaction(data)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    res := struct{TxHash string}{TxHash: tx.Hash().Hex()}
    js, err := json.Marshal(res)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write(js)
    
}

func transactionReceiptHandler(w http.ResponseWriter, r *http.Request, m *DVMService) {
    param := r.URL.Path[len("/tx/"):]
    txHash := common.HexToHash(param)
    log.Printf("in receipt handler(%s)\n", txHash.Hex())

    state, err := m.getState()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    tx, err := state.GetTransaction(txHash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
        return
	}

    receipt, err := state.GetReceipt(txHash)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

	

	signer := types.NewEIP155Signer(big.NewInt(1))
	from, err := types.Sender(signer, tx)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

	fields := map[string]interface{}{
		"root":              rpc.HexBytes(receipt.PostState),
		"transactionHash":   txHash,
		"from":              from,
		"to":                tx.To(),
		"gasUsed":           rpc.NewHexNumber(receipt.GasUsed),
		"cumulativeGasUsed": rpc.NewHexNumber(receipt.CumulativeGasUsed),
		"contractAddress":   nil,
		"logs":              receipt.Logs,
		"logsBloom":         receipt.Bloom,
	}
	if receipt.Logs == nil {
		fields["logs"] = []vm.Logs{}
	}
	// If the ContractAddress is 20 0x0 bytes, assume it is not a contract creation
	if receipt.ContractAddress != (common.Address{}) {
		fields["contractAddress"] = receipt.ContractAddress
	}
	
    js, err := json.Marshal(fields)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write(js)
}

func statsHandler(w http.ResponseWriter, r *http.Request, m *DVMService){
    platform := m.platform

    raftPlatfrom, ok := platform.(*DVMRaftPlatform)
    if !ok {
        http.Error(w, "Stats only applies to Raft platform", http.StatusMethodNotAllowed)
    }

    stats := raftPlatfrom.GetStats()

    js, err := json.Marshal(stats)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write(js)
}

//////////////////////////////////////////////////////////////////////////////

func prepareTransaction(args SendTxArgs, state *State, accMan *accounts.Manager )
 (*types.Transaction, error) {
	var err error   
	args, err = prepareSendTxArgs(args)
	if err != nil {
		return nil, err
	}

	if args.Nonce == nil {
		nonce := state.GetNonce(args.From)
		args.Nonce = rpc.NewHexNumber(nonce)
	}

	var tx *types.Transaction
	if args.To == nil {
		tx = types.NewContractCreation(args.Nonce.Uint64(), args.Value.BigInt(), args.Gas.BigInt(), args.GasPrice.BigInt(), common.FromHex(args.Data))
	} else {
		tx = types.NewTransaction(args.Nonce.Uint64(), *args.To, args.Value.BigInt(), args.Gas.BigInt(), args.GasPrice.BigInt(), common.FromHex(args.Data))
	}

	signer := types.NewEIP155Signer(big.NewInt(1))
	signature, err := accMan.SignEthereum(args.From, signer.Hash(tx).Bytes())
	if err != nil {
		return nil, err
	}
    signedTx, err := tx.WithSignature(signer, signature)
	if err != nil {
		return nil, err
	}

    return signedTx, nil
}

func prepareSendTxArgs(args SendTxArgs) (SendTxArgs, error) {
	if args.Gas == nil {
		args.Gas = rpc.NewHexNumber(defaultGas)
	}
	if args.GasPrice == nil {
		args.GasPrice = rpc.NewHexNumber(0)
	}
	if args.Value == nil {
		args.Value = rpc.NewHexNumber(0)
	}
	return args, nil
}

