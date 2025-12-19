package ws

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi/types"
	"github.com/jonsabados/saturdaysspinout/store"
	"github.com/rs/zerolog"
)

type Message struct {
	Action  string `json:"action"`
	Payload any    `json:"payload,omitempty"`
}

type APIGatewayManagementClient interface {
	PostToConnection(ctx context.Context, params *apigatewaymanagementapi.PostToConnectionInput, optFns ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.PostToConnectionOutput, error)
	DeleteConnection(ctx context.Context, params *apigatewaymanagementapi.DeleteConnectionInput, optFns ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.DeleteConnectionOutput, error)
}

type ConnectionLookup interface {
	GetConnectionsByDriver(ctx context.Context, driverID int64) ([]store.WebSocketConnection, error)
}

type Pusher struct {
	client           APIGatewayManagementClient
	connectionLookup ConnectionLookup
}

func NewPusher(client APIGatewayManagementClient, connectionLookup ConnectionLookup) *Pusher {
	return &Pusher{
		client:           client,
		connectionLookup: connectionLookup,
	}
}

// Push dispatches messages in a consistent format. When the connection is valid true, nil will be returned, but if the
// message could not be delivered due to the connection being disconnected false, nil will be returned.
func (p *Pusher) Push(ctx context.Context, connectionID string, actionType string, payload any) (bool, error) {
	fullPayload := Message{
		Action:  actionType,
		Payload: payload,
	}

	data, err := json.Marshal(fullPayload)
	if err != nil {
		return false, err
	}

	_, err = p.client.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
		ConnectionId: aws.String(connectionID),
		Data:         data,
	})
	if err != nil {
		var goneErr *types.GoneException
		if errors.As(err, &goneErr) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Disconnect closes a WebSocket connection.
func (p *Pusher) Disconnect(ctx context.Context, connectionID string) {
	logger := zerolog.Ctx(ctx)

	_, err := p.client.DeleteConnection(ctx, &apigatewaymanagementapi.DeleteConnectionInput{
		ConnectionId: aws.String(connectionID),
	})
	if err != nil {
		logger.Error().Err(err).Msg("failed to disconnect client")
	}
}

// Broadcast sends a message to all active connections for a given driver.
func (p *Pusher) Broadcast(ctx context.Context, driverID int64, actionType string, payload any) error {
	connections, err := p.connectionLookup.GetConnectionsByDriver(ctx, driverID)
	if err != nil {
		return err
	}

	for _, conn := range connections {
		if _, err := p.Push(ctx, conn.ConnectionID, actionType, payload); err != nil {
			return err
		}
	}

	return nil
}
