package ws

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi/types"
	"github.com/rs/zerolog"
)

type Message struct {
	Action  string `json:"action"`
	Payload any    `json:"payload,omitempty"`
}

type ConnectionDeleter interface {
	DeleteConnection(ctx context.Context, driverID int64, connectionID string) error
}

type Pusher struct {
	client    *apigatewaymanagementapi.Client
	connStore ConnectionDeleter
}

func NewPusher(client *apigatewaymanagementapi.Client, connStore ConnectionDeleter) *Pusher {
	return &Pusher{
		client:    client,
		connStore: connStore,
	}
}

// Push dispatches messages in a consistent format. driverID is used to clean up dynamo records during GoneExceptions,
// if the driverID is unknown 0 may be passed & cleanup steps on GoneErrors will be skipped.
func (p *Pusher) Push(ctx context.Context, driverID int64, connectionID string, actionType string, payload any) error {
	logger := zerolog.Ctx(ctx)

	fullPayload := Message{
		Action:  actionType,
		Payload: payload,
	}

	data, err := json.Marshal(fullPayload)
	if err != nil {
		return err
	}

	_, err = p.client.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
		ConnectionId: aws.String(connectionID),
		Data:         data,
	})
	if err != nil {
		var goneErr *types.GoneException
		// let's clean up closed connections when we know the driver and the key in dynamo
		if errors.As(err, &goneErr) && driverID != 0 {
			logger.Info().Str("connectionID", connectionID).Msg("connection gone, cleaning up")
			if err := p.connStore.DeleteConnection(ctx, driverID, connectionID); err != nil {
				logger.Error().Err(err).Str("connectionID", connectionID).Msg("failed to delete stale connection")
			}
		}
		return err
	}
	return nil
}

// Disconnect closes a connection, and attempts to clean up records of the connection in dynamo.
// Providing a 0 driverID will result in skipping dynamo cleanup.
func (p *Pusher) Disconnect(ctx context.Context, driverID int64, connectionID string) {
	logger := zerolog.Ctx(ctx)

	_, err := p.client.DeleteConnection(ctx, &apigatewaymanagementapi.DeleteConnectionInput{
		ConnectionId: aws.String(connectionID),
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to disconnect client")
	}
	if driverID != 0 {
		if err := p.connStore.DeleteConnection(ctx, driverID, connectionID); err != nil {
			logger.Error().Err(err).Str("connectionID", connectionID).Msg("failed to delete stale connection")
		}
	}
}
