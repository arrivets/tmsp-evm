package tmsp-evm

type Config struct {
    DataDir         string
    TendermintDir   string
    SeedPeers       []string
    APIAddr         string
}

type Platform struct {
    service Service
    state   State
    config  Config
}

func NewPlatform(config Config) (*Platform, error) {
    service := NewService(config.DataDir, config.APIAddr)
    state := new(State)
    return &Platform{sevice:service, state:state}, nil
}

func (p *Platform) Run() error {
    if err := p.state.Init(p); err != nil {return err}
	if err := p.service.Init(p); err != nil {return err}
    p.service.Run()
    return nil
}