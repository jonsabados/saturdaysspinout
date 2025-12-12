package ws

import (
	"context"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/rs/zerolog"
)

type RouteHandler interface {
	HandleRequest(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error)
}

type RouteHandlerFunc func(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error)

func (r RouteHandlerFunc) HandleRequest(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	return r(ctx, request)
}

// AuthenticatedMessage is a message that includes the driver ID for auth verification
type AuthenticatedMessage struct {
	Action   string `json:"action"`
	DriverID int64  `json:"driverId"`
}

type Handler struct {
	authHandler RouteHandler
	pingHandler RouteHandler
}

func NewHandler(authHandler RouteHandler, pingHandler RouteHandler) *Handler {
	return &Handler{authHandler: authHandler, pingHandler: pingHandler}
}

func (h *Handler) Handle(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	routeKey := request.RequestContext.RouteKey
	connectionID := request.RequestContext.ConnectionID

	logger := zerolog.Ctx(ctx).With().
		Str("routeKey", routeKey).
		Str("connectionID", connectionID).
		Logger()
	ctx = logger.WithContext(ctx)

	logger.Debug().Msg("handling websocket event")

	switch routeKey {
	case "$connect":
		return h.handleConnect(ctx, request)
	case "$disconnect":
		return h.handleDisconnect(ctx, request)
	case "auth":
		return h.authHandler.HandleRequest(ctx, request)
	case "pingRequest":
		return h.pingHandler.HandleRequest(ctx, request)
	case "$default":
		return h.handleDefault(ctx, request)
	default:
		logger.Warn().Msg("unhandled route")
		return events.APIGatewayProxyResponse{StatusCode: http.StatusBadRequest}, nil
	}
}

func (h *Handler) handleConnect(ctx context.Context, _ events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().Msg("websocket connected, awaiting auth")
	return events.APIGatewayProxyResponse{StatusCode: http.StatusOK}, nil
}

func (h *Handler) handleDisconnect(ctx context.Context, _ events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Info().Msg("websocket disconnected")
	return events.APIGatewayProxyResponse{StatusCode: http.StatusOK}, nil
}

func (h *Handler) handleDefault(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger := zerolog.Ctx(ctx)
	logger.Warn().Interface("request", request).Msg("default action called... this probably shouldn't be happening")
	return events.APIGatewayProxyResponse{StatusCode: http.StatusNotFound}, nil
}
