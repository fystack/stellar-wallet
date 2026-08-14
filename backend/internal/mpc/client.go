package mpc

import (
	"github.com/fystack/mpcium/pkg/client"
	"github.com/fystack/mpcium/pkg/event"
	"github.com/fystack/mpcium/pkg/types"
	"github.com/nats-io/nats.go"
)

type KeygenResult struct {
	WalletID   string
	PublicKey  []byte
	Successful bool
	Error      string
}

type SignResult struct {
	TransactionID string
	Signature     []byte
	Successful    bool
	Error         string
}

type Callbacks struct {
	OnKeygen func(KeygenResult)
	OnSign   func(SignResult)
}

type Client struct {
	client client.MPCClient
	nats   *nats.Conn
}

func New(natsURL, keyPath string, callbacks Callbacks) (*Client, error) {
	connection, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}
	signer, err := client.NewLocalSigner(types.EventInitiatorKeyTypeEd25519, client.LocalSignerOptions{
		KeyPath: keyPath,
	})
	if err != nil {
		connection.Close()
		return nil, err
	}
	mpcClient := client.NewMPCClient(client.Options{NatsConn: connection, Signer: signer})
	result := &Client{client: mpcClient, nats: connection}

	if err := mpcClient.OnWalletCreationResult(func(eventResult event.KeygenResultEvent) {
		if callbacks.OnKeygen != nil {
			callbacks.OnKeygen(KeygenResult{
				WalletID:   eventResult.WalletID,
				PublicKey:  eventResult.EDDSAPubKey,
				Successful: eventResult.ResultType == event.ResultTypeSuccess,
				Error:      eventResult.ErrorReason,
			})
		}
	}); err != nil {
		connection.Close()
		return nil, err
	}
	if err := mpcClient.OnSignResult(func(eventResult event.SigningResultEvent) {
		if callbacks.OnSign != nil {
			callbacks.OnSign(SignResult{
				TransactionID: eventResult.TxID,
				Signature:     eventResult.Signature,
				Successful:    eventResult.ResultType == event.ResultTypeSuccess,
				Error:         eventResult.ErrorReason,
			})
		}
	}); err != nil {
		connection.Close()
		return nil, err
	}
	return result, nil
}

func (c *Client) CreateWallet(walletID string) error {
	return c.client.CreateWallet(walletID)
}

func (c *Client) Sign(walletID, transactionID, network string, payload []byte) error {
	return c.client.SignTransaction(newSignMessage(walletID, transactionID, network, payload))
}

func (c *Client) Close() {
	if c.nats != nil {
		c.nats.Close()
	}
}

func newSignMessage(walletID, transactionID, network string, payload []byte) *types.SignTxMessage {
	return &types.SignTxMessage{
		KeyType:             types.KeyTypeEd25519,
		WalletID:            walletID,
		NetworkInternalCode: network,
		TxID:                transactionID,
		Tx:                  payload,
	}
}
