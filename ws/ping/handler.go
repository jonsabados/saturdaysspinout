package ping

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/jonsabados/saturdaysspinout/store"
	"github.com/jonsabados/saturdaysspinout/ws"
	"github.com/rs/zerolog"
)

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type Pusher interface {
	Push(ctx context.Context, driverID int64, connectionID string, actionType string, payload any) error
	Disconnect(ctx context.Context, driverID int64, connectionID string)
}

type ConnectionStore interface {
	GetConnection(ctx context.Context, driverID int64, connectionID string) (*store.WebSocketConnection, error)
}

func NewHandler(pusher Pusher, connectionStore ConnectionStore) ws.RouteHandler {
	return ws.RouteHandlerFunc(func(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
		logger := zerolog.Ctx(ctx)
		connectionID := request.RequestContext.ConnectionID

		var msg ws.AuthenticatedMessage
		if err := json.Unmarshal([]byte(request.Body), &msg); err != nil {
			logger.Warn().Err(err).Msg("failed to parse ping request")
			if err := pusher.Push(ctx, 0, connectionID, "pong", Response{Success: false, Message: "invalid payload"}); err != nil {
				logger.Error().Err(err).Msg("error pushing message")
			}
			return events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}, nil
		}

		if msg.DriverID == 0 {
			logger.Warn().Msg("missing driverId in ping request")
			if err := pusher.Push(ctx, 0, connectionID, "pong", Response{Success: false, Message: "missing driverId"}); err != nil {
				logger.Error().Err(err).Msg("error pushing message")
			}
			return events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}, nil
		}

		// Verify connection is authenticated for this driver
		conn, err := connectionStore.GetConnection(ctx, msg.DriverID, connectionID)
		if err != nil {
			logger.Error().Err(err).Msg("failed to get connection")
			return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
		}
		if conn == nil {
			logger.Warn().Int64("driverId", msg.DriverID).Msg("connection not found for driver, disconnecting")
			if err := pusher.Push(ctx, 0, connectionID, "pong", Response{Success: false, Message: "not authenticated"}); err != nil {
				logger.Error().Err(err).Msg("error pushing message")
			}
			pusher.Disconnect(ctx, msg.DriverID, connectionID)
			return events.APIGatewayProxyResponse{StatusCode: http.StatusForbidden}, nil
		}

		if err := pusher.Push(ctx, msg.DriverID, connectionID, "pong", Response{Success: true, Message: "pong"}); err != nil {
			logger.Error().Err(err).Msg("error pushing message")
			return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
		}

		return events.APIGatewayProxyResponse{StatusCode: http.StatusOK}, nil
	})
}
