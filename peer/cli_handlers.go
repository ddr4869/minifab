package peer

import (
	"github.com/ddr4869/minifab/common/logger"
	"github.com/pkg/errors"
)

// CLIHandlers CLI 명령어 핸들러들을 관리하는 구조체
type CLIHandlers struct {
	peer          *Peer
	ordererClient *OrdererClient
}

// NewCLIHandlers CLI 핸들러 생성
func NewCLIHandlers(peer *Peer, ordererClient *OrdererClient) *CLIHandlers {
	return &CLIHandlers{
		peer:          peer,
		ordererClient: ordererClient,
	}
}

// HandleChannelCreate 채널 생성 명령어 처리
func (h *CLIHandlers) HandleChannelCreate(channelName string, ordererAddress string) error {
	// 채널 생성 (새로운 로직 사용)
	if err := h.peer.GetChannelManager().CreateChannel(channelName, "SampleConsortium", ordererAddress); err != nil {
		return errors.Wrap(err, "failed to create channel")
	}
	logger.Infof("Channel %s created successfully", channelName)
	return nil
}

// HandleChannelJoin 채널 참여 명령어 처리
func (h *CLIHandlers) HandleChannelJoin(channelName string) error {
	if err := h.peer.JoinChannel(channelName); err != nil {
		return errors.Wrap(err, "failed to join channel")
	}
	logger.Infof("Successfully joined channel %s", channelName)
	return nil
}

// HandleChannelList 채널 목록 조회 명령어 처리
func (h *CLIHandlers) HandleChannelList() error {
	channels := h.peer.GetChannelManager().ListChannels()
	if len(channels) == 0 {
		logger.Info("No channels found")
		return nil
	}
	logger.Info("Available channels:")
	for _, channel := range channels {
		logger.Infof("- %s", channel)
	}
	return nil
}

// HandleTransaction 트랜잭션 처리 명령어
func (h *CLIHandlers) HandleTransaction(channelName string, payload []byte) error {
	// 트랜잭션 생성 및 제출
	tx, err := h.peer.SubmitTransaction(channelName, payload)
	if err != nil {
		return errors.Wrap(err, "failed to submit transaction")
	}
	logger.Infof("Transaction submitted successfully: %s", tx.ID)

	// Orderer에 트랜잭션 제출
	if err := h.ordererClient.SubmitTransaction(tx); err != nil {
		return errors.Wrap(err, "failed to submit transaction to orderer")
	}

	return nil
}
