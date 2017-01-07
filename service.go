package tmspevm

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/gorilla/mux"
	"github.com/tendermint/go-logger"
	"github.com/tendermint/log15"
)

const defaultGas = uint64(90000)

type Service struct {
	sync.Mutex
	platform       *Platform
	dataDir        string
	apiAddr        string
	accountManager *accounts.Manager
	log            log15.Logger
}

func NewService(dataDir, apiAddr string) *Service {
	return &Service{
		dataDir: dataDir,
		apiAddr: apiAddr,
		log:     logger.New("module", "service")}
}

func (m *Service) Init(platform *Platform) error {
	m.platform = platform
	return nil
}

func (m *Service) Run() {
	m.checkErr(m.makeAccountManager())

	m.checkErr(m.unlockAccounts())

	m.checkErr(m.createGenesisAccounts())

	m.log.Info("serving api...")
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
		Alloc AccountMap
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
		if err := m.accountManager.Unlock(account, "x"); err != nil {
			return err
		}
	}
	return nil
}

func (m *Service) getState() (*State, error) {
	return m.platform.GetState(), nil
}

func (m *Service) serveAPI() {
	router := mux.NewRouter()
	router.HandleFunc("/accounts", m.makeHandler(accountsHandler)).Methods("GET")
	router.HandleFunc("/tx", m.makeHandler(transactionHandler)).Methods("POST")
	router.HandleFunc("/tx/{tx_hash}", m.makeHandler(transactionReceiptHandler)).Methods("GET")
	http.ListenAndServe(m.apiAddr, router)
}

func (m *Service) makeHandler(fn func(http.ResponseWriter, *http.Request, *Service)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m.Lock()
		fn(w, r, m)
		m.Unlock()
	}
}

func (m *Service) checkErr(err error) {
	if err != nil {
		m.log.Error("ERROR", err)
		os.Exit(1)
	}
}
