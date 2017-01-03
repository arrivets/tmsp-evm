package tmsp-evm

package dvm

import (
    "os"
    "fmt"
    "path/filepath"
    "encoding/json"
    "io/ioutil"
    "math/big"
    "log"
    "net/http"
    "sync"

    "github.com/gorilla/mux"
    "github.com/ethereum/go-ethereum/accounts"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/rpc"

)

const defaultGas = uint64(90000)

type Service struct {
    sync.Mutex
    platform Platform
    dataDir string
    apiAddr string
    accountManager *accounts.Manager
}

func NewService(dataDir, apiAddr string) *Service{
    return &Service{dataDir: dataDir, apiAddr: apiAddr}
}

func (m *Service) Init(platform Platform) error {
    m.platform = platform
    return nil
}

func (m *Service) Run() {
    checkErr(m.makeAccountManager())

    checkErr(m.unlockAccounts())

    checkErr(m.createGenesisAccounts())
    
    log.Println("serving api...")
    m.serveAPI()
}

func (m *Service) makeAccountManager() error {
	scryptN := accounts.StandardScryptN
	scryptP := accounts.StandardScryptP
	
	keydir := filepath.Join(m.dataDir, "keystore")

	if err := os.MkdirAll(keydir, 0700); err != nil {
		return err
	}

	m.accountManager = accounts.NewManager(keydir, scryptN, scryptP)

    return nil
}

func (m *Service) createGenesisAccounts() error {
    genesisFile := filepath.Join(m.dataDir, "genesis.json")

    contents, err := ioutil.ReadFile(genesisFile)
	if err != nil {
		return err
	}

    var genesis struct {
		Alloc       AccountMap
	}

	if err := json.Unmarshal(contents, &genesis); err != nil {
		return err
	}
    state, err := m.getState()
    if err != nil {
        return err
    }
    if err := state.CreateAccounts(genesis.Alloc); err != nil {
        return err
    }
    return nil
}

func (m *Service) unlockAccounts() error {    
    accs := m.accountManager.Accounts()
    for _, account := range accs {
        if err := m.accountManager.Unlock(account,"x"); err != nil {
            return err
        }
    }
    return nil
}

func (m *Service) getState() (*State, error) {
    return m.platform.GetState()
}

func (m *Service) serveAPI() {
    router := mux.NewRouter()
    router.HandleFunc("/accounts", m.makeHandler(accountsHandler)).Methods("GET")
    router.HandleFunc("/tx", m.makeHandler(transactionHandler)).Methods("POST")
    router.HandleFunc("/tx/{tx_hash}", m.makeHandler(transactionReceiptHandler)).Methods("GET")
    router.HandleFunc("/stats", m.makeHandler(statsHandler)).Methods("GET")
    http.ListenAndServe(m.apiAddr, router)
}

func (m *Service) makeHandler(fn func (http.ResponseWriter, *http.Request, *Service)) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        m.Lock()
        fn(w, r, m)
        m.Unlock()
    }
}

func checkErr(err error) {
    if err != nil {
        log.Fatal("ERROR:", err)
    }
}




