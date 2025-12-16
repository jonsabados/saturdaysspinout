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

type APIGatewayManagementClient interface {
	PostToConnection(ctx context.Context, params *apigatewaymanagementapi.PostToConnectionInput, optFns ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.PostToConnectionOutput, error)
	DeleteConnection(ctx context.Context, params *apigatewaymanagementapi.DeleteConnectionInput, optFns ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.DeleteConnectionOutput, error)
}

type Pusher struct {
	client APIGatewayManagementClient
}

func NewPusher(client APIGatewayManagementClient) *Pusher {
	return &Pusher{
		client: client,
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
