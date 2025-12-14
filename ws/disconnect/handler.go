package disconnect

import (
	"context"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/jonsabados/saturdaysspinout/ws"
	"github.com/rs/zerolog"
)

type ConnectionStore interface {
	GetDriverIDByConnection(ctx context.Context, connectionID string) (*int64, error)
	DeleteConnection(ctx context.Context, driverID int64, connectionID string) error
}

func NewHandler(connStore ConnectionStore) ws.RouteHandler {
	return ws.RouteHandlerFunc(func(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
		logger := zerolog.Ctx(ctx)
		connectionID := request.RequestContext.ConnectionID

		driverID, err := connStore.GetDriverIDByConnection(ctx, connectionID)
		if err != nil {
			logger.Err(err).Msg("error looking up driver for connection")
			return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, nil
		}

		if driverID == nil {
			logger.Warn().Str("connection", connectionID).Msg("connection not found during disconnect")
			return events.APIGatewayProxyResponse{StatusCode: http.StatusOK}, nil
		}

		err = connStore.DeleteConnection(ctx, *driverID, connectionID)
		if err != nil {
			logger.Err(err).Msg("error deleting connection")
			return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, nil
		}

		return events.APIGatewayProxyResponse{StatusCode: http.StatusOK}, nil
	})
}
