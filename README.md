# TMSP-EVM
A lightweight EVM application for the Tendermint Socket Protocol. 

TMSP-EVM connects a Tendermint node to an instance of the Ethereum Virtual Machine (EVM).  
Ethereum transactions are ordered in the Tendermint blockchain and fed to the EVM  
via the TMSP interface. The EVM wrapper interprets and processes transactions  
sequentially to update its underlying state. An API service runs in parallel to handle  
private accounts and expose Ethereum functionality.  

The difference with the similar [Ethermint](http://github.com/tendermint/ethermint) project is that Ethermint is a wrapper  
around a full Geth node while TMSP-EVM only wraps the EVM and its state. 

## Design

```
                =============================================
============    =  ===============         ===============  =       
=          =    =  = Service     =         = State App   =  =
=  Client  <---------->          = <------ =             =  =
=          =    =  = -API        =         = -EVM        =  =
============    =  = -Acc Mann   =         = -Trie       =  =
                =  =             =         = -Database   =  =
                =  ===============         ===============  =
                =             |             ^               =
                ==============|=============|================
                              |Txs          |Txs
                ==============|=============|================
                = Platform    v             |               =
                =           ===================             =                                             
                =           = TMSP Server     =             = 
                =           ===================             =
                =              ^     ^     ^                =
                =              |   (TMSP)  |                =
                =              v     v     v                =
                =           ===================             =
                =           = TMSP Client     =             =
                =      =============================        =  
                =      = Tendermint Core           =        =
                =      =============================        =  
                =                   ^                       =
                ====================|========================  
                                    |
                                    |
                                    v
                                Consensus

```

## Dependencies

The first thing to do after cloning this repo is to get the appropriate depencies.  
We use [Glide](http://github.com/Masterminds/glide).  
```bash
sudo add-apt-repository ppa:masterminds/glide && sudo apt-get update
sudo apt-get install glide
```
Then inside the project folder:
```bash
glide install
```
This will download all the depencies and put them in the vendor folder.

## Usage

```
USAGE:
   tmsp-evm [global options] command [command options] [arguments...]

GLOBAL OPTIONS:
   --datadir "/home/<user>/.tmsp-evm"  Data directory for the databases and keystore
   --node_laddr value                  IP:Port to bind Tendermint consensus daemon on (default: "tcp://0.0.0.0:46656")
   --log_level value                   Tendermint log level (default: "info")
   --seeds value                       Comma delimited host:port seed nodes
   --no_fast_sync                      Disable fast blockchain syncing
   --skip_upnp                         Skip UPNP configuration
   --addr value                        TMSP app listen address (default: "tcp://0.0.0.0:46658")
   --apiaddr value                     IP:Port to bind API on (default: ":8080")
   --help, -h                          show help
   --version, -v                       print the version

```

## Configuration

The application writes data and reads configuration from the directory specified  
by the --datadir flag. The directory structure **MUST** be as follows:
```
host:~/.tmsp-evm$ tree
.
├── eth
│   ├── genesis.json
│   └── keystore
│       ├── [Ethereum key file]
│       ├── ...
│       ├── ...
│       ├── [Ethereum key file]
└── tendermint
    ├── config.toml
    ├── genesis.json
    ├── priv_validator.json
```
Notice that Ethereum and Tendermint use different genesis files.  
The Ethereum genesis file defines Ethereum accounts and is stripped of all   
the Ethereum POW blockchain stuff. The Tendermint genesis file   
defines Tendermint validators.  

Example Ethereum genesis.json defining two account:
```json
{
   "alloc": {
        "629007eb99ff5c3539ada8a5800847eacfc25727": {
            "balance": "1337000000000000000000"
        },
        "e32e14de8b81d8d3aedacb1868619c74a68feab0": {
            "balance": "1337000000000000000000"
        }
   }
}
```
The private keys for the above addresses should then reside in the keystore folder:
```
host:~/.tmsp-evm/eth/keystore$ tree
.
├── UTC--2016-02-01T16-52-27.910165812Z--629007eb99ff5c3539ada8a5800847eacfc25727
├── UTC--2016-02-01T16-52-28.021010343Z--e32e14de8b81d8d3aedacb1868619c74a68feab0
```

Example Tendermint config.toml:  
```
proxy_app = "tcp://127.0.0.1:46658"
moniker = "node1"
node_laddr = "tcp://0.0.0.0:46656"
seeds = ""
fast_sync = true
db_backend = "leveldb"
log_level = "notice"
rpc_laddr = "tcp://0.0.0.0:46657"
```

Example Tendermint genesis.json defining a single validator:  
```json
{
        "app_hash": "",
        "chain_id": "test-chain",
        "genesis_time": "0001-01-01T00:00:00.000Z",
        "validators": [
                {
                        "amount": 10,
                        "name": "",
                        "pub_key": [
                                1,
                                "DFF2D4103ABF699E3F9E9DBA97BB9B1E0E9BD87CE826E5319C5FB89FBFB661ED"
                        ]
                }
        ]
}
```
The validator's private key resides in the priv_validator.json file:
```
{
        "address": "3A91EFD9149D23DF69599631826E3412171BB972",
        "last_height": 0,
        "last_round": 0,
        "last_step": 0,
        "priv_key": [
                1,
                "80D58FFBAB861709449AA8C91E0AAD52FC4C7EDBDBA530031B30BFB930ADA635DFF2D4103ABF699E3F9E9DBA97BB9B1E0E9BD87CE826E5319C5FB89FBFB661ED"
        ],
        "pub_key": [
                1,
                "DFF2D4103ABF699E3F9E9DBA97BB9B1E0E9BD87CE826E5319C5FB89FBFB661ED"
        ]
}
```

**Needless to say you should not reuse these addresses and private keys**

## API
The Service exposes an API at the address specified by the --apiaddr flag for  
clients to interact with Ethereum.

### List accounts 
example:
```bash
host:~$ curl http://localhost:8080/accounts -s | json_pp
{
   "Accounts" : [
      {
         "Address" : "0x629007eb99ff5c3539ada8a5800847eacfc25727",
         "Balance" : "1337000000000000000000"
      },
      {
         "Address" : "0xe32e14de8b81d8d3aedacb1868619c74a68feab0",
         "Balance" : "1337000000000000000000"
      }
   ]
}
```

### Create Ethereum transactions
example: Send Ether between accounts  
```bash
host:~$ curl -X POST http://localhost:8080/tx -d '{"from":"0x629007eb99ff5c3539ada8a5800847eacfc25727","to":"0xe32e14de8b81d8d3aedacb1868619c74a68feab0","value":6666}' -s | json_pp
{
   "TxHash" : "0xeeeed34877502baa305442e3a72df094cfbb0b928a7c53447745ff35d50020bf"
}

```

### Get Transaction receipt
example:
```bash
host:~$ curl http://localhost:8080/tx/0xeeeed34877502baa305442e3a72df094cfbb0b928a7c53447745ff35d50020bf -s | json_pp
{
   "to" : "0xe32e14de8b81d8d3aedacb1868619c74a68feab0",
   "root" : "0xc8f90911c9280651a0cd84116826d31773e902e48cb9a15b7bb1e7a6abc850c5",
   "gasUsed" : "0x5208",
   "from" : "0x629007eb99ff5c3539ada8a5800847eacfc25727",
   "transactionHash" : "0xeeeed34877502baa305442e3a72df094cfbb0b928a7c53447745ff35d50020bf",
   "logs" : [],
   "cumulativeGasUsed" : "0x5208",
   "contractAddress" : null,
   "logsBloom" : "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
}

```

Then check accounts again to see that the balances have changed:
```bash
{
   "Accounts" : [
      {
         "Address" : "0x629007eb99ff5c3539ada8a5800847eacfc25727",
         "Balance" : "1336999999999999993334"
      },
      {
         "Address" : "0xe32e14de8b81d8d3aedacb1868619c74a68feab0",
         "Balance" : "1337000000000000006666"
      }
   ]
}
```
## Docker Testnet
The docker folder contains a Dockerfile to package the tmsp-evm application along  
with some scripts to bootstrap a testnet of four nodes.

```bash
cp docker
./build-docker
./run-testnet
./stop-testnet
```
The node APIs can be reached on localhost at ports 8081 to 8084 for testing








