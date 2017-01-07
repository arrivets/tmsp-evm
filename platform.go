package tmspevm

import (
	"encoding/hex"

	cfg "github.com/tendermint/go-config"
	"github.com/tendermint/go-logger"
	rpcclient "github.com/tendermint/go-rpc/client"
	"github.com/tendermint/log15"
	"github.com/tendermint/tendermint/node"
	core_types "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tmsp/server"
)

type Config struct {
	EthDir  string
	ApiAddr string

	TmConfig cfg.Config
}

type Platform struct {
	service *Service
	state   *State
	client  *rpcclient.ClientURI
	config  Config
	log     log15.Logger
}

func NewPlatform(config Config) (*Platform, error) {
	service := NewService(config.EthDir, config.ApiAddr)
	state := new(State)
	client := rpcclient.NewClientURI(config.TmConfig.GetString("rpc_laddr"))
	log := logger.New("module", "platform")
	return &Platform{service: service,
		state:  state,
		client: client,
		config: config,
		log:    log}, nil
}

func (p *Platform) Run() error {
	if err := p.state.Init(p); err != nil {
		return err
	}

	proxyAddr := p.config.TmConfig.GetString("proxy_app")
	_, err := server.NewServer(proxyAddr, "socket", p.state)
	if err != nil {
		return err
	}

	go node.RunNode(p.config.TmConfig)

	if err := p.service.Init(p); err != nil {
		return err
	}
	p.service.Run()

	return nil
}

func (p *Platform) CreateTransaction(tx []byte) error {
	var result core_types.TMResult
	params := map[string]interface{}{
		"tx": hex.EncodeToString(tx),
	}
	_, err := p.client.Call("broadcast_tx_sync", params, &result)
	return err

}

func (p *Platform) GetState() *State {
	return p.state
}
