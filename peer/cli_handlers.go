package peer

import (
	"fmt"

	"github.com/ddr4869/minifab/common/logger"
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
		return fmt.Errorf("failed to create channel: %v", err)
	}
	logger.Infof("Channel %s created successfully", channelName)

	// Orderer에 채널 생성 알림 (실패해도 계속 진행)
	if err := h.ordererClient.CreateChannel(channelName); err != nil {
		logger.Warnf("Failed to notify orderer about channel creation: %v", err)
	} else {
		logger.Infof("Orderer notified about channel %s creation", channelName)
	}

	return nil
}

// HandleChannelJoin 채널 참여 명령어 처리
func (h *CLIHandlers) HandleChannelJoin(channelName string) error {
	if err := h.peer.JoinChannel(channelName); err != nil {
		return fmt.Errorf("failed to join channel: %v", err)
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

// HandleTransaction 트랜잭션 제출 명령어 처리
func (h *CLIHandlers) HandleTransaction(channelName string, payload []byte) error {
	// 트랜잭션 생성 및 제출
	tx, err := h.peer.SubmitTransaction(channelName, payload)
	if err != nil {
		return fmt.Errorf("failed to submit transaction: %v", err)
	}
	logger.Infof("Transaction submitted successfully: %s", tx.ID)
	logger.Infof("Transaction identity: %s", tx.Identity)

	// 트랜잭션 검증
	if err := h.peer.ValidateTransaction(tx); err != nil {
		logger.Warnf("Transaction validation failed: %v", err)
		return err
	}

	// Orderer에 트랜잭션 제출
	if err := h.ordererClient.SubmitTransaction(tx); err != nil {
		return fmt.Errorf("failed to submit transaction to orderer: %v", err)
	}

	logger.Infof("Transaction submitted to orderer successfully: %s", tx.ID)
	return nil
}
