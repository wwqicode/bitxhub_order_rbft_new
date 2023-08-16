package main

import (
	"sort"
	"time"

	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/pkg/order"
	"github.com/sirupsen/logrus"
	"github.com/ultramesh/rbft"
	"github.com/ultramesh/rbft/mempool"
	"github.com/ultramesh/rbft/rbftpb"
)

type RBFTConfig struct {
	Rbft RBFT
}

type RBFT struct {
	SetSize          int           `mapstructure:"set_size"`
	BatchSize        uint64        `mapstructure:"batch_size"`
	PoolSize         uint64        `mapstructure:"pool_size"`
	CheckInterval    time.Duration `mapstructure:"check_interval"`
	ToleranceTime    time.Duration `mapstructure:"tolerance_time"`
	BatchMemLimit    bool          `mapstructure:"batch_mem_limit"`
	BatchMaxMem      uint64        `mapstructure:"batch_max_mem"`
	VCPeriod         uint64        `mapstructure:"vc_period"`
	GetBlockByHeight func(height uint64) (*pb.Block, error)
	Timeout
}

type Timeout struct {
	SyncState        time.Duration `mapstructure:"sync_state"`
	SyncInterval     time.Duration `mapstructure:"sync_interval"`
	Recovery         time.Duration `mapstructure:"recovery"`
	FirstRequest     time.Duration `mapstructure:"first_request"`
	Batch            time.Duration `mapstructure:"batch"`
	Request          time.Duration `mapstructure:"request"`
	NullRequest      time.Duration `mapstructure:"null_request"`
	ViewChange       time.Duration `mapstructure:"viewchange"`
	ResendViewChange time.Duration `mapstructure:"resend_viewchange"`
	CleanViewChange  time.Duration `mapstructure:"clean_viewchange"`
	Update           time.Duration `mapstructure:"update"`
	Set              time.Duration `mapstructure:"set"`
}

func defaultRbftConfig() rbft.Config {
	return rbft.Config{
		SetSize:                 1000,
		IsNew:                   false,
		K:                       10,
		LogMultiplier:           4,
		VCPeriod:                0,
		CheckPoolTimeout:        100 * time.Second,
		SyncStateTimeout:        3 * time.Second,
		SyncStateRestartTimeout: 40 * time.Second,
		RecoveryTimeout:         10 * time.Second,
		FirstRequestTimeout:     30 * time.Second,
		BatchTimeout:            200 * time.Millisecond,
		RequestTimeout:          6 * time.Second,
		NullRequestTimeout:      9 * time.Second,
		NewViewTimeout:          1 * time.Second,
		VcResendTimeout:         8 * time.Second,
		CleanVCTimeout:          60 * time.Second,
		UpdateTimeout:           4 * time.Second,
		SetTimeout:              100 * time.Millisecond,
	}
}

func generateRbftConfig(repoRoot string, config *order.Config) (rbft.Config, error) {
	readConfig, err := readConfig(repoRoot)
	if err != nil {
		return rbft.Config{}, nil
	}

	defaultConfig := defaultRbftConfig()
	defaultConfig.ID = config.ID
	defaultConfig.Logger = &Logger{config.Logger}
	defaultConfig.Peers, err = generateRbftPeers(config)
	if err != nil {
		return rbft.Config{}, err
	}

	defaultConfig.CheckPoolTimeout = readConfig.Rbft.CheckInterval
	defaultConfig.SetSize = readConfig.Rbft.SetSize
	defaultConfig.VCPeriod = readConfig.Rbft.VCPeriod
	defaultConfig.Applied = config.Applied
	defaultConfig.GetBlockByHeight = config.GetBlockByHeight

	defaultConfig.SyncStateTimeout = readConfig.Rbft.Timeout.SyncState
	defaultConfig.SyncStateRestartTimeout = readConfig.Rbft.Timeout.SyncInterval
	defaultConfig.RecoveryTimeout = readConfig.Rbft.Timeout.Recovery
	defaultConfig.FirstRequestTimeout = readConfig.Rbft.Timeout.FirstRequest
	defaultConfig.BatchTimeout = readConfig.Rbft.Timeout.Batch
	defaultConfig.RequestTimeout = readConfig.Rbft.Timeout.Request
	defaultConfig.NullRequestTimeout = readConfig.Rbft.Timeout.NullRequest
	defaultConfig.NewViewTimeout = readConfig.Rbft.Timeout.ViewChange
	defaultConfig.VcResendTimeout = readConfig.Rbft.Timeout.ResendViewChange
	defaultConfig.CleanVCTimeout = readConfig.Rbft.Timeout.CleanViewChange
	defaultConfig.UpdateTimeout = readConfig.Rbft.Timeout.Update
	defaultConfig.SetTimeout = readConfig.Rbft.Timeout.Set
	defaultConfig.IsNew = config.IsNew

	mempoolConf := mempool.Config{
		ID:              config.ID,
		Logger:          defaultConfig.Logger,
		BatchSize:       readConfig.Rbft.BatchSize,
		PoolSize:        readConfig.Rbft.PoolSize,
		BatchMemLimit:   readConfig.Rbft.BatchMemLimit,
		BatchMaxMem:     readConfig.Rbft.BatchMaxMem,
		ToleranceTime:   readConfig.Rbft.ToleranceTime,
		GetAccountNonce: config.GetAccountNonce,
	}
	defaultConfig.PoolConfig = mempoolConf
	return defaultConfig, nil
}

func generateRbftPeers(config *order.Config) ([]*rbftpb.Peer, error) {
	return sortPeers(config.Nodes)
}

func sortPeers(nodes map[uint64]*pb.VpInfo) ([]*rbftpb.Peer, error) {
	peers := make([]*rbftpb.Peer, 0, len(nodes))
	for id, vpInfo := range nodes {
		vpIngoBytes, err := vpInfo.Marshal()
		if err != nil {
			return nil, err
		}
		peers = append(peers, &rbftpb.Peer{Id: id, Context: vpIngoBytes})
	}
	sort.Slice(peers, func(i, j int) bool {
		return peers[i].Id < peers[j].Id
	})
	return peers, nil
}

type Logger struct {
	logrus.FieldLogger
}

func (lg *Logger) Critical(v ...interface{}) {
	lg.Info(v...)
}

func (lg *Logger) Criticalf(format string, v ...interface{}) {
	lg.Infof(format, v...)
}

func (lg *Logger) Notice(v ...interface{}) {
	lg.Info(v...)
}

func (lg *Logger) Noticef(format string, v ...interface{}) {
	lg.Infof(format, v...)
}
