package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/pkg/order"
	"github.com/meshplus/bitxhub/pkg/peermgr/mock_peermgr"
	"github.com/sirupsen/logrus"
	"github.com/ultramesh/rbft"
	"github.com/ultramesh/rbft/mempool"
	mockexternal "github.com/ultramesh/rbft/mock/mock_external"
	"github.com/ultramesh/rbft/rbftpb"
)

var storagePath = path.Join(getProjectPath(), "/rbft/testdata/storage")

func mockOrder(ctrl *gomock.Controller) order.Order {
	order, _ := NewNode(withID(), withIsNew(), withRepoRoot(), withStoragePath(),
		withLogger(), withNodes(), withApplied(), withDigest(), withPeerManager(ctrl), withGetAccountNonceFunc())
	return order
}

func mockMiniPeerManager(ctrl *gomock.Controller) *mock_peermgr.MockPeerManager {
	mock := mock_peermgr.NewMockPeerManager(ctrl)
	mock.EXPECT().Broadcast(gomock.Any()).Return(nil).AnyTimes()
	mock.EXPECT().AsyncSend(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mock.EXPECT().AddNode(gomock.Any(), gomock.Any()).Return().AnyTimes()
	mock.EXPECT().DelNode(gomock.Any()).Return().AnyTimes()
	mock.EXPECT().UpdateRouter(gomock.Any(), gomock.Any()).Return(false).AnyTimes()
	block := constructBlock("block2", uint64(2))
	blockBytes, _ := block.Marshal()
	res := &pb.Message{Data: blockBytes}
	mock.EXPECT().Send(gomock.Any(), gomock.Any()).Return(res, nil).AnyTimes()
	nodes := make(map[uint64]*pb.VpInfo)
	nodes[1] = &pb.VpInfo{Id: uint64(1)}
	nodes[2] = &pb.VpInfo{Id: uint64(2)}
	nodes[3] = &pb.VpInfo{Id: uint64(3)}
	mock.EXPECT().Peers().Return(nodes).AnyTimes()
	mock.EXPECT().Disconnect(gomock.Any()).Return().AnyTimes()
	return mock
}

func withID() order.Option {
	return func(config *order.Config) {
		config.ID = uint64(1)
	}
}

func withIsNew() order.Option {
	return func(config *order.Config) {
		config.IsNew = false
	}
}

func withRepoRoot() order.Option {
	return func(config *order.Config) {
		config.RepoRoot = "./testdata/"
	}
}

func withStoragePath() order.Option {
	return func(config *order.Config) {
		config.StoragePath = storagePath
	}
}

func withLogger() order.Option {
	return func(config *order.Config) {
		config.Logger = log.NewWithModule("order")
	}
}

func withPeerManager(ctrl *gomock.Controller) order.Option {
	return func(config *order.Config) {
		config.PeerMgr = mockMiniPeerManager(ctrl)
	}
}

func withGetAccountNonceFunc() order.Option {
	return func(config *order.Config) {
		config.GetAccountNonce = func(address *types.Address) uint64 {
			return 0
		}
	}
}

func withNodes() order.Option {
	nodes := make(map[uint64]*pb.VpInfo)
	nodes[1] = &pb.VpInfo{Id: uint64(1)}
	nodes[2] = &pb.VpInfo{Id: uint64(2)}
	nodes[3] = &pb.VpInfo{Id: uint64(3)}
	return func(config *order.Config) {
		config.Nodes = nodes
	}
}

func withApplied() order.Option {
	return func(config *order.Config) {
		config.Applied = uint64(1)
	}
}

func withDigest() order.Option {
	return func(config *order.Config) {
		config.Digest = "digest"
	}
}

func mockNode(ctrl *gomock.Controller) *Node {
	logger := log.NewWithModule("order")
	rbftConf := mockRBFTConfig(ctrl, logger)
	etcdNode, _ := rbft.NewNode(rbftConf)
	store, _ := NewStorage(storagePath)
	blockC := make(chan *pb.CommitEvent, 1024)
	ctx, cancel := context.WithCancel(context.Background())
	stack, _ := NewStack(store, mockOrderConfig(logger, ctrl), blockC, cancel, rbftConf.IsNew)
	node := &Node{
		id:      uint64(1),
		n:       etcdNode,
		logger:  logger,
		stack:   stack,
		blockC:  blockC,
		ctx:     ctx,
		txCache: newTxCache(0, 0, logger),
	}
	stack.applyConfChange = node.n.ApplyConfChange
	return node
}

func mockRBFTConfig(ctrl *gomock.Controller, logger *logrus.Entry) rbft.Config {
	peerSet := make([]*rbftpb.Peer, 0)
	peerSet = append(peerSet, &rbftpb.Peer{Id: uint64(1)}, &rbftpb.Peer{Id: uint64(2)}, &rbftpb.Peer{Id: uint64(3)})
	external := mockexternal.NewMockMinimalExternal(ctrl)
	conf := rbft.Config{
		ID:                      uint64(1),
		IsNew:                   false,
		K:                       10,
		LogMultiplier:           4,
		SetSize:                 25,
		SetTimeout:              100 * time.Millisecond,
		BatchTimeout:            500 * time.Millisecond,
		RequestTimeout:          6 * time.Second,
		NullRequestTimeout:      9 * time.Second,
		VcResendTimeout:         10 * time.Second,
		CleanVCTimeout:          60 * time.Second,
		NewViewTimeout:          8 * time.Second,
		FirstRequestTimeout:     30 * time.Second,
		SyncStateTimeout:        1 * time.Second,
		SyncStateRestartTimeout: 10 * time.Second,
		RecoveryTimeout:         10 * time.Second,
		UpdateTimeout:           4 * time.Second,
		CheckPoolTimeout:        3 * time.Minute,
		Peers:                   peerSet,
		External:                external,
		Logger:                  &Logger{logger},
		PoolConfig:              mempool.NewMockMempoolConfig(),
	}
	return conf
}

func mockOrderConfig(logger logrus.FieldLogger, ctrl *gomock.Controller) *order.Config {
	nodes := make(map[uint64]*pb.VpInfo)
	priv := genPrivKey()
	account, _ := priv.PublicKey().Address()
	nodes[1] = &pb.VpInfo{Id: uint64(1), Account: account.String()}
	nodes[2] = &pb.VpInfo{Id: uint64(2)}
	nodes[3] = &pb.VpInfo{Id: uint64(3)}
	conf := &order.Config{
		ID:               uint64(1),
		Nodes:            nodes,
		IsNew:            false,
		Logger:           logger,
		PeerMgr:          mockMiniPeerManager(ctrl),
		PrivKey:          priv,
		GetChainMetaFunc: getChainMetaFunc,
	}
	return conf
}

func constructBlock(blockHashStr string, height uint64) *pb.Block {
	from := make([]byte, 0)
	strLen := len(blockHashStr)
	for i := 0; i < 32; i++ {
		from = append(from, blockHashStr[i%strLen])
	}
	fromStr := hex.EncodeToString(from)
	blockHash := types.NewHashByStr(fromStr)
	header := &pb.BlockHeader{
		Number:     height,
		ParentHash: blockHash,
		Timestamp:  time.Now().UnixNano(),
	}
	return &pb.Block{
		BlockHash:   blockHash,
		BlockHeader: header,
	}
}

// getProjectPath get complete path for project
func getProjectPath() string {
	goPath := os.Getenv("GOPATH")
	goRoot := os.Getenv("GOROOT")
	if strings.Contains(goPath, ":") {
		goPathes := strings.Split(goPath, ":")
		var realPathes []string
		for _, p := range goPathes {
			if p == goRoot {
				continue
			}
			realPathes = append(realPathes, p)
		}
		goPath = realPathes[0]
	}
	return goPath + "/src/github.com/meshplus/bitxhub-order-rbft"
}

func cleanData() {
	err := os.RemoveAll(storagePath)
	if err != nil {
		fmt.Printf("error:%s", err)
		return
	}
}

func genPrivKey() crypto.PrivateKey {
	privKey, _ := asym.GenerateKeyPair(crypto.Secp256k1)
	return privKey
}

func getChainMetaFunc() *pb.ChainMeta {
	block := constructBlock("block1", uint64(1))
	return &pb.ChainMeta{Height: uint64(1), BlockHash: block.BlockHash}
}
