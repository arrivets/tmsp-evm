package tmsp-evm

import (
    "math/big"
    
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

type AccountMap map[string] struct {
    Code    string
    Storage map[string]string
    Balance string
}
type JsonAccount struct{
        Address string
        Balance *big.Int 
    }

type JsonAccountList struct {
    Accounts []JsonAccount
}
// SendTxArgs represents the arguments to sumbit a new transaction into the transaction pool.
type SendTxArgs struct {
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *rpc.HexNumber  `json:"gas"`
	GasPrice *rpc.HexNumber  `json:"gasPrice"`
	Value    *rpc.HexNumber  `json:"value"`
	Data     string          `json:"data"`
	Nonce    *rpc.HexNumber  `json:"nonce"`
}

