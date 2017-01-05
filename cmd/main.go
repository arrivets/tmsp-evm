package main

import (
    "os"
    "os/user"
	"path/filepath"
	"runtime"
    "log"

	tevm "github.com/arrivets/tmsp-evm"
    "gopkg.in/urfave/cli.v1"
    
	"github.com/ethereum/go-ethereum/cmd/utils"
    "github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"

	cfg "github.com/tendermint/go-config"
    tmcfg "github.com/tendermint/tendermint/config/tendermint"
    tmlog "github.com/tendermint/go-logger"
)

var (
    DataDirFlag = utils.DirectoryFlag{
		Name:  "datadir",
		Usage: "Data directory for the databases and keystore",
		Value: utils.DirectoryString{DefaultDataDir()},
	}
    NodeAddressFlag = cli.StringFlag{
		Name:  "node_laddr",
		Usage: "IP:Port to bind Tendermint consensus daemon on",
		Value: "tcp://0.0.0.0:46656",
	}
	LogLevelFlag = cli.StringFlag{
		Name:  "log_level",
		Usage: "Tendermint log level",
        Value: "info",
	}
    SeedsFlag = cli.StringFlag{
        Name:  "seeds",
        Value: "",
        Usage: "Comma delimited host:port seed nodes",
    }
	SyncFlag =	cli.BoolFlag{
        Name:  "no_fast_sync",
        Usage: "Disable fast blockchain syncing",
    }
	UpnpFlag =	cli.BoolFlag{
        Name:  "skip_upnp",
        Usage: "Skip UPNP configuration",
    }
	TmspAddressFlag =  cli.StringFlag{
        Name:  "addr",
        Usage: "TMSP app listen address",
        Value: "tcp://0.0.0.0:46658",
    }
	APIAddrFlag = cli.StringFlag{
		Name: "apiaddr",
		Usage: "IP:Port to bind API on",
		Value: ":8080",
	}
) 

func main() {
    app := makeApp()
    log.Fatal(app.Run(os.Args))
}

func makeApp() *cli.App {
	app := cli.NewApp()
    app.Name = "tmsp-evm"
    app.Usage = "this is the usage"
    app.Flags = []cli.Flag{ 
        DataDirFlag, 
        NodeAddressFlag,
        LogLevelFlag,
		SeedsFlag,
        SyncFlag,
        UpnpFlag,
        TmspAddressFlag, 
        APIAddrFlag }
    app.Action = run
	
	app.After = func(ctx *cli.Context) error {
		logger.Flush()
		return nil
	}

	// logging
	glog.SetV(logger.Detail)
	glog.SetToStderr(true)
	return app
}

func run(ctx *cli.Context) error {
	ethDir := filepath.Join(ctx.GlobalString(DataDirFlag.Name),"eth")
	config := tevm.Config{
		EthDir: ethDir,
		ApiAddr: ctx.GlobalString(APIAddrFlag.Name),
        TmConfig: getTendermintConfig(ctx),
	}

    platform, err := tevm.NewPlatform(config)
    if err != nil {
        return err
    }
    return(platform.Run())	
}

func getTendermintConfig(ctx *cli.Context) cfg.Config {
	tmDir := filepath.Join(ctx.GlobalString(DataDirFlag.Name), "tendermint")
	os.Setenv("TMROOT", tmDir)
	config := tmcfg.GetConfig("")
	config.Set("node_laddr", ctx.GlobalString(NodeAddressFlag.Name))
	config.Set("seeds", ctx.GlobalString(SeedsFlag.Name))
	config.Set("fast_sync", ctx.GlobalBool(SyncFlag.Name))
	config.Set("skip_upnp", ctx.GlobalBool(UpnpFlag.Name))
	config.Set("proxy_app", ctx.GlobalString(TmspAddressFlag.Name))
	config.Set("log_level", ctx.GlobalString(LogLevelFlag.Name))
	tmlog.SetLogLevel(config.GetString("log_level"))
	return config
}

// DefaultDataDir is the default data directory to use for the databases and other
// persistence requirements.
func DefaultDataDir() string {
	// Try to place the data folder in the user's home dir
	home := homeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "TMSPEVM")
		} else if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Roaming", "TMSPEVM")
		} else {
			return filepath.Join(home, ".tmsp-evm")
		}
	}
	// As we cannot guess a stable location, return empty and handle later
	return ""
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}