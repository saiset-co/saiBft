package internal

import (
	"github.com/iamthe1whoknocks/bft/models"
	"github.com/iamthe1whoknocks/bft/utils"
	"go.uber.org/zap"
)

// unput data for testing purposes

// save test tx (for testing purposes)
func (s *InternalService) saveTestTx(saiBtcAddress, storageToken, saiP2PAddress string) {
	testTxMsg := &models.TransactionMessage{
		Votes: 0,
		Tx: &models.Tx{
			Type:          models.TransactionMsgType,
			SenderAddress: s.BTCkeys.Address,
			Message:       "test tx message",
		},
	}

	testTxHash, err := testTxMsg.Tx.GetHash()
	if err != nil {
		s.GlobalService.Logger.Fatal("processing - hash test tx error", zap.Error(err))
	}

	testTxMsg.Tx.MessageHash = testTxHash
	testTxMsg.MessageHash = testTxHash

	resp, err := utils.SignMessage(testTxMsg, saiBtcAddress, s.BTCkeys.Private)
	if err != nil {
		s.GlobalService.Logger.Fatal("processing - sign test tx error", zap.Error(err))
	}
	testTxMsg.Tx.SenderSignature = resp.Signature

	err, _ = s.Storage.Put("MessagesPool", testTxMsg, storageToken)
	if err != nil {
		s.GlobalService.Logger.Fatal("processing - put test tx msg", zap.Error(err))
	}

	bcErr := s.broadcastMsg(testTxMsg.Tx, saiP2PAddress)
	if bcErr != nil {
		s.GlobalService.Logger.Fatal("processing - broadcast test tx msg", zap.Error(err))
	}
	s.GlobalService.Logger.Sugar().Debugf("test tx message saved") //DEBUG
}

// save test consensusMsg (for testing purposes)
func (s *InternalService) saveTestConsensusMsg(saiBtcAddress, storageToken, senderAddress string) {
	testConsensusMsg := &models.ConsensusMessage{
		Type:          models.ConsensusMsgType,
		SenderAddress: senderAddress,
		BlockNumber:   3,
		Round:         7,
		Messages:      []string{"0060ee497708e7d9a8428802a6651b93847dca9a0217d05ad67a5a1be7d49223"},
	}

	testConsensusHash, err := testConsensusMsg.GetHash()
	if err != nil {
		s.GlobalService.Logger.Fatal("processing - hash test consensus error", zap.Error(err))
	}

	testConsensusMsg.Hash = testConsensusHash

	resp, err := utils.SignMessage(testConsensusMsg, saiBtcAddress, s.BTCkeys.Private)
	if err != nil {
		s.GlobalService.Logger.Fatal("processing - sign test consensus error", zap.Error(err))
	}
	testConsensusMsg.Signature = resp.Signature

	err, _ = s.Storage.Put("ConsensusPool", testConsensusMsg, storageToken)
	if err != nil {
		s.GlobalService.Logger.Fatal("processing - put test consensus msg", zap.Error(err))
	}

	s.GlobalService.Logger.Sugar().Debugf("test consensus message saved") //DEBUG

}
