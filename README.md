# tmkv

kv store built on tendermint


## building

note: we are using latest tendermint
- `go get`
- `make`

## running 
`TMHOME=~/.tendermint && ./build/tmkv ~/.tendermint/config/config.toml`



## config
you can generate config using

`tendermint init validator`


### update genesis
- Add validators in `genesis` validators section using `priv_validator_key.json`

```
➜  tmkv git:(master) ✗ cat ~/.tendermint/config/priv_validator_key.json 
{
  "address": "327D57A4C74DE54C9F21625129642DF3F52D1E4E",
  "pub_key": {
    "type": "tendermint/PubKeyEd25519",
    "value": "UDCnz0K6nbnJndaXr4vGLAWYpPoPk7mmFyLSB9XdIfU="
  },
  "priv_key": {
    "type": "tendermint/PrivKeyEd25519",
    "value": "/p9iN9OTmuyXWa2SlKi3YSsbtWBd9i27YNux8/j71itQMKfPQrqducmd1pevi8YsBZik+g+TuaYXItIH1d0h9Q=="
  }
}%                                                                                                                           
➜  tmkv git:(master) ✗ cat ~/.tendermint/config/genesis.json           
{
  "genesis_time": "2021-07-06T22:32:26.60389942Z",
  "chain_id": "test-chain-Tb1Cha",
  "initial_height": "0",
  "consensus_params": {
    "block": {
      "max_bytes": "22020096",
      "max_gas": "-1"
    },
    "evidence": {
      "max_age_num_blocks": "100000",
      "max_age_duration": "172800000000000",
      "max_bytes": "1048576"
    },
    "validator": {
      "pub_key_types": [
        "ed25519"
      ]
    },
    "version": {
      "app_version": "0"
    }
  },
  "validators": [
    {
      "address": "327D57A4C74DE54C9F21625129642DF3F52D1E4E",
      "pub_key": {
        "type": "tendermint/PubKeyEd25519",
        "value": "UDCnz0K6nbnJndaXr4vGLAWYpPoPk7mmFyLSB9XdIfU="
      },
      "power": "10",
      "name": ""
    }
  ],
  "app_hash": ""
}%          
```