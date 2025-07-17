package cli

import (
	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/peer/client"
	"github.com/ddr4869/minifab/peer/core"
	"github.com/pkg/errors"
)

// Handlers CLI 명령어 핸들러들을 관리하는 구조체
type Handlers struct {
	peer          *core.Peer
	ordererClient *client.OrdererClient
}

// NewHandlers CLI 핸들러 생성
func NewHandlers(peer *core.Peer, ordererClient *client.OrdererClient) *Handlers {
	return &Handlers{
		peer:          peer,
		ordererClient: ordererClient,
	}
}

// HandleChannelCreate 채널 생성 명령어 처리
func (h *Handlers) HandleChannelCreate(channelName string, ordererAddress string) error {
	// 채널 생성 (orderer를 통한 새로운 로직 사용)
	if err := h.peer.CreateChannel(channelName, h.ordererClient); err != nil {
		return errors.Wrap(err, "failed to create channel")
	}
	logger.Infof("Channel %s created successfully", channelName)
	return nil
}

// HandleChannelCreateWithProfile 프로파일을 사용한 채널 생성 명령어 처리
func (h *Handlers) HandleChannelCreateWithProfile(channelName, profileName string) error {
	// 채널 생성 (orderer를 통한 프로파일 기반 로직 사용)
	if err := h.peer.CreateChannelWithProfile(channelName, profileName, h.ordererClient); err != nil {
		return errors.Wrap(err, "failed to create channel with profile")
	}
	logger.Infof("Channel %s created successfully with profile %s", channelName, profileName)
	return nil
}

// HandleChannelJoin 채널 참여 명령어 처리
func (h *Handlers) HandleChannelJoin(channelName string) error {
	if err := h.peer.JoinChannel(channelName, h.ordererClient); err != nil {
		return errors.Wrap(err, "failed to join channel")
	}
	logger.Infof("Successfully joined channel %s", channelName)
	return nil
}

// HandleChannelJoinWithProfile 프로파일을 사용한 채널 참여 명령어 처리
func (h *Handlers) HandleChannelJoinWithProfile(channelName, profileName string) error {
	if err := h.peer.JoinChannelWithProfile(channelName, profileName, h.ordererClient); err != nil {
		return errors.Wrap(err, "failed to join channel with profile")
	}
	logger.Infof("Successfully joined channel %s with profile %s", channelName, profileName)
	return nil
}

// HandleChannelList 채널 목록 조회 명령어 처리
func (h *Handlers) HandleChannelList() error {
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
func (h *Handlers) HandleTransaction(channelName string, payload []byte) error {
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
