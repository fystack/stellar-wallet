package main

import (
	"encoding/hex"
	"log"

	"github.com/fystack/mpcium/pkg/client"
	"github.com/fystack/mpcium/pkg/event"
	"github.com/fystack/mpcium/pkg/types"
	"github.com/nats-io/nats.go"
)

// mpc wraps the mpcium client and bridges keygen/sign results back into our DB + SSE.
type mpc struct {
	client client.MPCClient
	nats   *nats.Conn
	srv    *server
}

func newMPC(s *server, natsURL, keyPath, clientID string) (*mpc, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}
	signer, err := client.NewLocalSigner(types.EventInitiatorKeyTypeEd25519, client.LocalSignerOptions{
		KeyPath: keyPath,
	})
	if err != nil {
		return nil, err
	}
	c := client.NewMPCClient(client.Options{
		NatsConn: nc,
		Signer:   signer,
		ClientID: clientID,
	})

	m := &mpc{client: c, nats: nc, srv: s}

	if err := c.OnWalletCreationResult(m.onKeygen); err != nil {
		return nil, err
	}
	if err := c.OnSignResult(m.onSign); err != nil {
		return nil, err
	}
	return m, nil
}

// CreateWallet kicks off a distributed keygen ceremony for walletID.
func (m *mpc) CreateWallet(walletID string) error {
	return m.client.CreateWallet(walletID)
}

// Sign asks the cluster to threshold-sign tx bytes for a wallet.
func (m *mpc) Sign(walletID, txID, network string, tx []byte) error {
	return m.client.SignTransaction(&types.SignTxMessage{
		KeyType:             types.KeyTypeEd25519,
		WalletID:            walletID,
		NetworkInternalCode: network,
		TxID:                txID,
		Tx:                  tx,
	})
}

func (m *mpc) onKeygen(evt event.KeygenResultEvent) {
	if evt.ResultType != event.ResultTypeSuccess {
		log.Printf("keygen failed wallet=%s reason=%s", evt.WalletID, evt.ErrorReason)
		m.srv.db.Exec(`UPDATE wallets SET status = 'failed' WHERE id = ?`, evt.WalletID)
		return
	}

	// Look up which chain this wallet was created for.
	var chain, userID string
	if err := m.srv.db.QueryRow(`SELECT chain, user_id FROM wallets WHERE id = ?`, evt.WalletID).
		Scan(&chain, &userID); err != nil {
		return
	}
	address := deriveAddress(chain, evt.EDDSAPubKey)
	pubHex := hex.EncodeToString(evt.EDDSAPubKey)

	m.srv.db.Exec(
		`UPDATE wallets SET address = ?, pubkey = ?, status = 'ready' WHERE id = ?`,
		address, pubHex, evt.WalletID)

	if w, ok := m.srv.walletByID(evt.WalletID); ok {
		m.srv.hub.publishWallet(userID, w)
	}
}

func (m *mpc) onSign(evt event.SigningResultEvent) {
	// Load the pending transaction (unscoped — we only have the tx id here).
	txns := m.srv.txnsWhere(`id = ?`, evt.TxID)
	if len(txns) == 0 {
		return
	}
	tx := txns[0]

	var userID string
	m.srv.db.QueryRow(`SELECT user_id FROM transactions WHERE id = ?`, evt.TxID).Scan(&userID)

	if evt.ResultType != event.ResultTypeSuccess {
		log.Printf("sign failed tx=%s reason=%s", evt.TxID, evt.ErrorReason)
		reason := evt.ErrorReason
		if reason == "" {
			reason = "MPC signing failed"
		}
		m.srv.db.Exec(`UPDATE transactions SET status = 'failed', error = ? WHERE id = ?`, reason, evt.TxID)
		m.publishTxnByID(userID, evt.TxID)
		return
	}

	sig := hex.EncodeToString(evt.Signature)
	m.srv.db.Exec(`UPDATE transactions SET signature = ? WHERE id = ?`, sig, evt.TxID)

	// Stellar: attach the signature and broadcast to Horizon.
	if tx.EnvelopeXDR != "" {
		w, _ := m.srv.walletByID(tx.WalletID)
		hash, err := stellarSubmit(tx.EnvelopeXDR, w.Pubkey, evt.Signature)
		if err != nil {
			log.Printf("broadcast failed tx=%s: %v", evt.TxID, err)
			m.srv.db.Exec(`UPDATE transactions SET status = 'failed', error = ? WHERE id = ?`,
				"broadcast: "+err.Error(), evt.TxID)
			m.publishTxnByID(userID, evt.TxID)
			return
		}
		m.srv.db.Exec(`UPDATE transactions SET status = 'broadcast', tx_hash = ? WHERE id = ?`, hash, evt.TxID)
		m.publishTxnByID(userID, evt.TxID)
		m.srv.db.Exec(`UPDATE transactions SET status = 'confirmed' WHERE id = ?`, evt.TxID)
		m.publishTxnByID(userID, evt.TxID)
		go m.srv.refreshWallet(tx.WalletID)
		return
	}

	// Non-broadcast chains: signature is the terminal result.
	m.srv.db.Exec(`UPDATE transactions SET status = 'confirmed' WHERE id = ?`, evt.TxID)
	m.publishTxnByID(userID, evt.TxID)
}

func (m *mpc) publishTxnByID(userID, txID string) {
	if txns := m.srv.txnsWhere(`id = ?`, txID); len(txns) > 0 {
		m.srv.hub.publishTxn(userID, txns[0])
	}
}
