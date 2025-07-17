package channel

import (
	"errors"

	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/peer/core"
)

// InitializeChannelManager는 peer에 channelManager를 설정합니다
func InitializeChannelManager(peer *core.Peer) error {
	if peer == nil {
		return errors.New("peer instance is nil")
	}

	// Channel Manager 생성
	channelManager := NewManager()

	// Peer에 Channel Manager 설정
	peer.SetChannelManager(channelManager)

	logger.Infof("✅ Channel manager initialized and set to peer")
	return nil
}

// EnsureChannelManagerInitialized는 peer의 channelManager가 초기화되어 있는지 확인하고, 없으면 초기화합니다
func EnsureChannelManagerInitialized(peer *core.Peer) error {
	if peer == nil {
		return errors.New("peer instance is nil")
	}

	// 이미 channelManager가 설정되어 있는지 확인
	if peer.GetChannelManager() != nil {
		return nil // 이미 초기화됨
	}

	// 초기화 필요
	return InitializeChannelManager(peer)
}
