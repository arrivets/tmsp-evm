package tmspevm

import (
    "math/big"
    "fmt"
    "bytes"
    "sync"
    "log"

    "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core"
    ethTypes "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
    "github.com/ethereum/go-ethereum/rlp"
    "github.com/ethereum/go-ethereum/params"

    tmspTypes "github.com/tendermint/tmsp/types"

)

var (
    gasLimit = big.NewInt(1000000000000000000)
    txMetaSuffix   = []byte{0x01}
	receiptsPrefix = []byte("receipts-")
    MIPMapLevels = []uint64{1000000, 500000, 100000, 50000, 1000}
)

type State struct {
    platform    *Platform

    db          ethdb.Database
    commitMutex sync.Mutex
    statedb     *state.StateDB
    was         *WriteAheadState

    signer      ethTypes.Signer
    chainConfig params.ChainConfig //vm.env is still tightly coupled with chainConfig
	vmConfig    vm.Config
}

// write ahead state, updated with each AppendTx
// and reset on Commit
type WriteAheadState struct {
        state  *state.StateDB

        txIndex      int
        transactions []*ethTypes.Transaction
        receipts     ethTypes.Receipts
        allLogs      vm.Logs

        totalUsedGas *big.Int
        gp           *core.GasPool
}

func (s *State) Init(platform *Platform) error {
    var err error
    s.platform = platform
    s.db, err  =  ethdb.NewMemDatabase() //ephemeral database
    if err != nil {
        return err
    }
    state, err := state.New(common.Hash{}, s.db)
    if err != nil {
        return err
    }

    s.statedb = state  
    s.resetWAS(state.Copy())    

    s.signer = ethTypes.NewEIP155Signer(big.NewInt(1))
    s.chainConfig = params.ChainConfig{big.NewInt(1), new(big.Int), new(big.Int), true, new(big.Int), common.Hash{}, new(big.Int), new(big.Int)}
    s.vmConfig = vm.Config{Tracer: vm.NewStructLogger(nil)}
    return nil
}

// Applications --------------------------------------------------------------

// Return application info
func (s *State) Info() (info string){
    return "tmsp-evm" 
}

// Set application option (e.g. mode=mempool, mode=consensus)
func (s *State)	SetOption(key string, value string) (log string){
    return "not implemented"
}

// Append a tx
func (s *State)	AppendTx(tx []byte) tmspTypes.Result {
    s.commitMutex.Lock()
    defer s.commitMutex.Unlock()

    var t ethTypes.Transaction
    if err := rlp.Decode(bytes.NewReader(tx), &t); err != nil {
        return tmspTypes.ErrEncodingError       
    }
    msg, err := t.AsMessage(s.signer)
    if err != nil {
        return tmspTypes.NewError(tmspTypes.CodeType_InternalError, 
         fmt.Sprintf("AppendTx AsMessage: %v", err))
    }
    context := vm.Context{
        CanTransfer: core.CanTransfer,
        Transfer:    core.Transfer,
        GetHash:     func(uint64) common.Hash { return common.Hash{} },
        // Message information
        Origin:   msg.From(), 
        GasPrice: msg.GasPrice(),
    }
    // Environment provides information about external sources for the EVM
    // The Environment should never be reused and is not thread safe.
    vmenv := vm.NewEnvironment(context, s.was.state, &s.chainConfig, s.vmConfig)
    // Apply the transaction to the current state (included in the env)
    _, gas, err := core.ApplyMessage(vmenv, msg, s.was.gp)
    if err != nil {
        return tmspTypes.NewError(tmspTypes.CodeType_InternalError,
         fmt.Sprintf("AppendTx ApplyMessage: %v", err))
    }

    s.was.totalUsedGas.Add(s.was.totalUsedGas, gas)

    // Create a new receipt for the transaction, storing the intermediate root and gas used by the tx
    // based on the eip phase, we're passing wether the root touch-delete accounts.
    receipt := ethTypes.NewReceipt(s.statedb.IntermediateRoot(true).Bytes(), s.was.totalUsedGas)
    receipt.TxHash = t.Hash()
    receipt.GasUsed = new(big.Int).Set(gas)
    // if the transaction created a contract, store the creation address in the receipt.
    if msg.To() == nil {
        receipt.ContractAddress = crypto.CreateAddress(vmenv.Context.Origin, t.Nonce())
    }
    // Set the receipt logs and create a bloom for filtering
    receipt.Logs = s.was.state.GetLogs(t.Hash())
    receipt.Bloom = ethTypes.CreateBloom(ethTypes.Receipts{receipt})

    s.was.txIndex += 1
    s.was.transactions = append(s.was.transactions, &t)
    s.was.receipts = append(s.was.receipts, receipt)
    s.was.allLogs = append(s.was.allLogs, receipt.Logs...)

    return tmspTypes.OK
}

// Validate a tx for the mempool
func (s *State)	CheckTx(tx []byte) tmspTypes.Result {
    log.Printf("XXXX in CheckTx")
    var t ethTypes.Transaction
    if err := rlp.Decode(bytes.NewReader(tx), &t); err != nil {
        return tmspTypes.ErrEncodingError       
    }
    
    from, err := ethTypes.Sender(s.signer, &t)
    if err != nil {
        return tmspTypes.NewError(tmspTypes.CodeType_InternalError,
        fmt.Sprintf("CheckTx invalid sender: %v", err))
    }
    
    if s.was.state.GetNonce(from) > t.Nonce() {
        return tmspTypes.ErrBadNonce
    }

    // Check the transaction doesn't exceed the current
    // block limit gas.
    if (*big.Int)(s.was.gp).Cmp(t.Gas()) < 0 {
        return tmspTypes.NewError(tmspTypes.CodeType_InternalError,
        fmt.Sprintf("CheckTx gas limit: %v", err))
    }

    // Transactions can't be negative. This may never happen
    // using RLP decoded transactions but may occur if you create
    // a transaction using the RPC for example.
    if t.Value().Cmp(common.Big0) < 0 {
        return tmspTypes.NewError(tmspTypes.CodeType_InternalError,
        fmt.Sprintf("CheckTx negative value: %v", err))
    }

    // Transactor should have enough funds to cover the costs
    // cost == V + GP * GL
    if s.was.state.GetBalance(from).Cmp(t.Cost()) < 0 {
        return tmspTypes.ErrInsufficientFunds
    }

    //XXX: Check intinsic gas

    return tmspTypes.OK
}

// Query for state
func (s *State)	Query(query []byte) tmspTypes.Result {
    return tmspTypes.NewResultOK(nil,"not implemented")
}

// Return the application Merkle root hash
func (s *State)	Commit() tmspTypes.Result {
    s.commitMutex.Lock()
    defer s.commitMutex.Unlock()

    //commit all state changes to the database 
    hashArray, err := s.was.state.Commit(true)
    if err != nil {
        log.Printf("Error committing ethereum state trie: %v", err)
        return tmspTypes.ErrInternalError
    }
   
    // reset the write ahead state for the next block
    // with the latest eth state
    s.statedb = s.was.state
    s.resetWAS(s.statedb.Copy())

    return tmspTypes.NewResultOK(hashArray.Bytes(), "")
}

//----------------------------------------------------------------------------

// runs in Commit once we have the new state
func (s *State) resetWAS(state *state.StateDB) {
        s.was = &WriteAheadState{
                state:        state,
                txIndex:      0,
                totalUsedGas: big.NewInt(0),
                gp:           new(core.GasPool).AddGas(gasLimit),
        }
}

func (s *State) CreateAccounts(accounts AccountMap) error {
    s.commitMutex.Lock()
    defer s.commitMutex.Unlock()

    for addr, account := range accounts {
		address := common.HexToAddress(addr)
		s.was.state.AddBalance(address, common.String2Big(account.Balance))
        s.was.state.SetCode(address, common.Hex2Bytes(account.Code))
		for key, value := range account.Storage {
			s.was.state.SetState(address, common.HexToHash(key), common.HexToHash(value))
		}
	}
	_, err := s.was.state.Commit(true)
    if err != nil {
		return fmt.Errorf("cannot write state: %v", err)
	}

    s.statedb = s.was.state
    s.resetWAS(s.statedb.Copy())

    return nil
}

func (s *State) GetBalance(addr common.Address) *big.Int {
    return s.statedb.GetBalance(addr)
}

func (s *State) GetNonce(addr common.Address) uint64 {
    return s.statedb.GetNonce(addr)
}

func (s *State) GetTransaction(hash common.Hash) (*ethTypes.Transaction, error) {
	// Retrieve the transaction itself from the database
	data, err := s.db.Get(hash.Bytes())
	if err != nil {
		return nil, fmt.Errorf("get-transaction: %v",err)
	}
	var tx ethTypes.Transaction
	if err := rlp.DecodeBytes(data, &tx); err != nil {
		return nil, err
	}
	
	return &tx, nil
}

func (s *State) GetReceipt(txHash common.Hash) (*ethTypes.Receipt, error) {
	data, err := s.db.Get(append(receiptsPrefix, txHash[:]...))
	if err != nil {
		return nil, fmt.Errorf("get-receipt: %v",err)
	}
	var receipt ethTypes.ReceiptForStorage
	if err := rlp.DecodeBytes(data, &receipt); err != nil {
        return nil, err
    }
	
	return (*ethTypes.Receipt)(&receipt), nil
}

