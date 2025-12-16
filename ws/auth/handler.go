package auth

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/jonsabados/saturdaysspinout/auth"
	"github.com/jonsabados/saturdaysspinout/store"
	"github.com/jonsabados/saturdaysspinout/ws"
	"github.com/rs/zerolog"
)

type Request struct {
	Action string `json:"action"`
	Token  string `json:"token"`
}

type Response struct {
	Success      bool   `json:"success"`
	UserID       int64  `json:"userId,omitempty"`
	ConnectionID string `json:"connectionId,omitempty"`
	Error        string `json:"error,omitempty"`
}

type ConnectionStore interface {
	SaveConnection(ctx context.Context, conn store.WebSocketConnection) error
}

type Pusher interface {
	Push(ctx context.Context, connectionID string, actionType string, payload any) (bool, error)
	Disconnect(ctx context.Context, connectionID string)
}

type JWTValidator interface {
	ValidateToken(ctx context.Context, tokenString string) (*auth.SessionClaims, *auth.SensitiveClaims, error)
}

func NewHandler(validator JWTValidator, pusher Pusher, connStore ConnectionStore) ws.RouteHandler {
	return ws.RouteHandlerFunc(func(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
		logger := zerolog.Ctx(ctx)
		connectionID := request.RequestContext.ConnectionID

		var authMsg Request
		if err := json.Unmarshal([]byte(request.Body), &authMsg); err != nil {
			logger.Warn().Err(err).Msg("failed to parse auth message")
			_, err := pusher.Push(ctx, connectionID, "authResponse", Response{Success: false, Error: "invalid payload"})
			if err != nil {
				zerolog.Ctx(ctx).Err(err).Msg("error replying")
				return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, nil
			}
			return events.APIGatewayProxyResponse{StatusCode: http.StatusOK}, nil
		}

		if authMsg.Token == "" {
			logger.Warn().Msg("empty token")
			_, err := pusher.Push(ctx, connectionID, "authResponse", Response{Success: false, Error: "missing token"})
			if err != nil {
				zerolog.Ctx(ctx).Err(err).Msg("error replying")
				return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, nil
			}
			return events.APIGatewayProxyResponse{StatusCode: http.StatusOK}, nil
		}

		sessionClaims, _, err := validator.ValidateToken(ctx, authMsg.Token)
		if err != nil {
			logger.Warn().Err(err).Msg("invalid token")
			_, err := pusher.Push(ctx, connectionID, "authResponse", Response{Success: false, Error: "invalid token"})
			if err != nil {
				zerolog.Ctx(ctx).Err(err).Msg("error replying")
				return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, nil
			}
			// since auth failed lets kill the websocket connection, make em go away and come back
			pusher.Disconnect(ctx, connectionID)
			return events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}, nil
		}

		logger.Info().Int64("userID", sessionClaims.IRacingUserID).Str("userName", sessionClaims.IRacingUserName).Msg("authenticated websocket connection")

		err = connStore.SaveConnection(ctx, store.WebSocketConnection{
			DriverID:     sessionClaims.IRacingUserID,
			ConnectionID: connectionID,
		})
		if err != nil {
			logger.Error().Err(err).Msg("failed to save connection")
			_, err := pusher.Push(ctx, connectionID, "authResponse", Response{Success: false, Error: "internal error"})
			if err != nil {
				zerolog.Ctx(ctx).Err(err).Msg("error replying")
				return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, nil
			}
			// connections in a funky state now, so lets nuke it
			pusher.Disconnect(ctx, connectionID)
			return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
		}

		_, err = pusher.Push(ctx, connectionID, "authResponse", Response{Success: true, UserID: sessionClaims.IRacingUserID, ConnectionID: connectionID})
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Msg("error replying")
			return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, nil
		}
		return events.APIGatewayProxyResponse{StatusCode: http.StatusOK}, nil
	})
}
