package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/tendermint/tendermint/abci/server"
	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/log"
	tmnet "github.com/tendermint/tendermint/libs/net"
	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/privval"
	grpcprivval "github.com/tendermint/tendermint/privval/grpc"
	privvalproto "github.com/tendermint/tendermint/proto/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
)

var logger = log.MustNewDefaultLogger(log.LogFormatPlain, log.LogLevelInfo, false)

// main is the binary entrypoint.
func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %v <configfile>", os.Args[0])
		return
	}
	configFile := ""
	if len(os.Args) == 2 {
		configFile = os.Args[1]
	}

	if err := run(configFile); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

// run runs the application - basically like main() with error handling.
func run(configFile string) error {
	cfg, err := LoadConfig(configFile)
	if err != nil {
		return err
	}

	// Start remote signer (must start before node if running builtin).
	if cfg.PrivValServer != "" {
		if err = startSigner(cfg); err != nil {
			return err
		}
		if cfg.Protocol == "builtin" {
			time.Sleep(1 * time.Second)
		}
	}

	// Start app server.
	switch cfg.Protocol {
	case "socket", "grpc":
		err = startApp(cfg)
	case "builtin":

		err = startNode(cfg)
	default:
		err = fmt.Errorf("invalid protocol %q", cfg.Protocol)
	}
	if err != nil {
		return err
	}

	// Apparently there's no way to wait for the server, so we just sleep
	for {
		time.Sleep(1 * time.Hour)
	}
}

// startApp starts the application server, listening for connections from Tendermint.
func startApp(cfg *Config) error {
	app, err := NewApplication(cfg)
	if err != nil {
		return err
	}
	server, err := server.NewServer(cfg.Listen, cfg.Protocol, app)
	if err != nil {
		return err
	}
	err = server.Start()
	if err != nil {
		return err
	}
	logger.Info(fmt.Sprintf("Server listening on %v (%v protocol)", cfg.Listen, cfg.Protocol))
	return nil
}

// startNode starts a Tendermint node running the application directly. It assumes the Tendermint
// configuration is in $TMHOME/config/tendermint.toml.
//
// FIXME There is no way to simply load the configuration from a file, so we need to pull in Viper.
func startNode(cfg *Config) error {
	app, err := NewApplication(cfg)
	if err != nil {
		return err
	}

	tmcfg, nodeLogger, err := setupNode()
	if err != nil {
		return fmt.Errorf("failed to setup config: %w", err)
	}

	n, err := node.New(tmcfg,
		nodeLogger,
		proxy.NewLocalClientCreator(app),
		nil,
	)
	if err != nil {
		return err
	}
	return n.Start()
}

func startSeedNode(cfg *Config) error {
	tmcfg, nodeLogger, err := setupNode()
	if err != nil {
		return fmt.Errorf("failed to setup config: %w", err)
	}

	tmcfg.Mode = config.ModeSeed

	n, err := node.New(tmcfg, nodeLogger, nil, nil)
	if err != nil {
		return err
	}
	return n.Start()
}

// startSigner starts a signer server connecting to the given endpoint.
func startSigner(cfg *Config) error {
	filePV, err := privval.LoadFilePV(cfg.PrivValKey, cfg.PrivValState)
	if err != nil {
		return err
	}

	protocol, address := tmnet.ProtocolAndAddress(cfg.PrivValServer)
	var dialFn privval.SocketDialer
	switch protocol {
	case "tcp":
		dialFn = privval.DialTCPFn(address, 3*time.Second, ed25519.GenPrivKey())
	case "unix":
		dialFn = privval.DialUnixFn(address)
	case "grpc":
		lis, err := net.Listen("tcp", address)
		if err != nil {
			return err
		}
		ss := grpcprivval.NewSignerServer(cfg.ChainID, filePV, logger)

		s := grpc.NewServer()

		privvalproto.RegisterPrivValidatorAPIServer(s, ss)

		go func() { // no need to clean up since we remove docker containers
			if err := s.Serve(lis); err != nil {
				panic(err)
			}
		}()

		return nil
	default:
		return fmt.Errorf("invalid privval protocol %q", protocol)
	}

	endpoint := privval.NewSignerDialerEndpoint(logger, dialFn,
		privval.SignerDialerEndpointRetryWaitInterval(1*time.Second),
		privval.SignerDialerEndpointConnRetries(100))
	err = privval.NewSignerServer(endpoint, cfg.ChainID, filePV).Start()
	if err != nil {
		return err
	}

	logger.Info(fmt.Sprintf("Remote signer connecting to %v", cfg.PrivValServer))
	return nil
}

func setupNode() (*config.Config, log.Logger, error) {
	var tmcfg *config.Config

	home := os.Getenv("TMHOME")
	if home == "" {
		return nil, nil, errors.New("TMHOME not set")
	}

	viper.AddConfigPath(filepath.Join(home, "config"))
	viper.SetConfigName("config")

	if err := viper.ReadInConfig(); err != nil {
		return nil, nil, err
	}

	tmcfg = config.DefaultConfig()

	if err := viper.Unmarshal(tmcfg); err != nil {
		return nil, nil, err
	}

	tmcfg.SetRoot(home)

	if err := tmcfg.ValidateBasic(); err != nil {
		return nil, nil, fmt.Errorf("error in config file: %w", err)
	}

	nodeLogger, err := log.NewDefaultLogger(tmcfg.LogFormat, tmcfg.LogLevel, false)
	if err != nil {
		return nil, nil, err
	}

	return tmcfg, nodeLogger.With("module", "main"), nil
}
