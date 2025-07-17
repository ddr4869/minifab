package cli

import (
	"github.com/ddr4869/minifab/common/logger"
	"github.com/ddr4869/minifab/peer/channel"
	"github.com/ddr4869/minifab/peer/client"
	"github.com/ddr4869/minifab/peer/core"
	"github.com/pkg/errors"
)

var CliHandlers *Handlers

type Handlers struct {
	peer          *core.Peer
	ordererClient *client.OrdererClient
}

// NewHandlers CLI 핸들러 생성
func NewHandlers(peer *core.Peer, ordererClient *client.OrdererClient) *Handlers {
	CliHandlers = &Handlers{
		peer:          peer,
		ordererClient: ordererClient,
	}
	return CliHandlers
}

// ensureChannelManagerInitialized는 채널 관련 작업 전에 channelManager가 초기화되어 있는지 확인합니다
func (h *Handlers) ensureChannelManagerInitialized() error {
	return channel.EnsureChannelManagerInitialized(h.peer)
}

// HandleChannelCreate 채널 생성 명령어 처리 - 단순히 peer의 메서드를 호출
func (h *Handlers) HandleChannelCreate(channelName string, ordererAddress string) error {
	if err := h.ensureChannelManagerInitialized(); err != nil {
		return errors.Wrap(err, "failed to initialize channel manager")
	}
	return h.peer.CreateChannel(channelName, h.ordererClient)
}

// HandleChannelCreateWithProfile 프로파일을 사용한 채널 생성 명령어 처리 - 단순히 peer의 메서드를 호출
func (h *Handlers) HandleChannelCreateWithProfile(channelName, profileName string) error {
	if err := h.ensureChannelManagerInitialized(); err != nil {
		return errors.Wrap(err, "failed to initialize channel manager")
	}
	return h.peer.CreateChannelWithProfile(channelName, profileName, h.ordererClient)
}

// HandleChannelJoin 채널 참여 명령어 처리 - 단순히 peer의 메서드를 호출
func (h *Handlers) HandleChannelJoin(channelName string) error {
	if err := h.ensureChannelManagerInitialized(); err != nil {
		return errors.Wrap(err, "failed to initialize channel manager")
	}
	return h.peer.JoinChannel(channelName, h.ordererClient)
}

// HandleChannelJoinWithProfile 프로파일을 사용한 채널 참여 명령어 처리 - 단순히 peer의 메서드를 호출
func (h *Handlers) HandleChannelJoinWithProfile(channelName, profileName string) error {
	if err := h.ensureChannelManagerInitialized(); err != nil {
		return errors.Wrap(err, "failed to initialize channel manager")
	}
	return h.peer.JoinChannelWithProfile(channelName, profileName, h.ordererClient)
}

// HandleChannelList 채널 목록 조회 명령어 처리 - 단순히 peer의 메서드를 호출
func (h *Handlers) HandleChannelList() error {
	if err := h.ensureChannelManagerInitialized(); err != nil {
		return errors.Wrap(err, "failed to initialize channel manager")
	}

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
	if err := h.ensureChannelManagerInitialized(); err != nil {
		return errors.Wrap(err, "failed to initialize channel manager")
	}

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
