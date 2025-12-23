package sqs

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-xray-sdk-go/v2/xray"
	"github.com/rs/zerolog"
)

type HandlerFunc func(ctx context.Context, event events.SQSEvent) error

func WithReducedContextDeadline(h HandlerFunc, buffer time.Duration) HandlerFunc {
	return func(ctx context.Context, event events.SQSEvent) error {
		deadline, ok := ctx.Deadline()
		if !ok {
			zerolog.Ctx(ctx).Warn().Msg("no deadline present on context")
			return h(ctx, event)
		}
		newDeadline := deadline.Add(-buffer)
		if newDeadline.Before(time.Now()) {
			return fmt.Errorf("attempt to reduce deadline by more than possible, original: %q, new: %q", deadline, newDeadline)
		}
		zerolog.Ctx(ctx).Debug().Time("original", deadline).Time("new", newDeadline).Msg("reducing deadline")
		ctx, cancel := context.WithDeadline(ctx, newDeadline)
		defer cancel()
		return h(ctx, event)
	}
}

func WithPanicProtection(h HandlerFunc) HandlerFunc {
	return func(ctx context.Context, event events.SQSEvent) (err error) {
		defer func() {
			if e := recover(); e != nil {
				stack := debug.Stack()
				zerolog.Ctx(ctx).Error().
					Interface("panic", e).
					Bytes("stack", stack).
					Msg("recovered from panic")

				switch v := e.(type) {
				case string:
					err = fmt.Errorf("recovered from panic: %s", v)
				case error:
					err = fmt.Errorf("recovered from panic: %w", v)
				default:
					err = fmt.Errorf("recovered from panic: %v", v)
				}
			}
		}()
		err = h(ctx, event)
		return
	}
}

type VisibilityTimeoutComputer func(msg events.SQSMessage) int32

func LinearVisibilityTimeoutComputer(step time.Duration) VisibilityTimeoutComputer {
	return func(msg events.SQSMessage) int32 {
		receiveCount := 1
		if countStr, ok := msg.Attributes["ApproximateReceiveCount"]; ok {
			if parsed, err := fmt.Sscanf(countStr, "%d", &receiveCount); err != nil || parsed != 1 {
				receiveCount = 1
			}
		}
		return int32((receiveCount - 1) * int(step.Seconds()))
	}
}

type SQSClient interface {
	ChangeMessageVisibility(ctx context.Context, params *sqs.ChangeMessageVisibilityInput, optFns ...func(*sqs.Options)) (*sqs.ChangeMessageVisibilityOutput, error)
	GetQueueUrl(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error)
}

func parseQueueARN(arn string) (accountID, queueName string, err error) {
	parts := strings.Split(arn, ":")
	if len(parts) != 6 {
		return "", "", fmt.Errorf("invalid SQS ARN format: %s", arn)
	}
	return parts[4], parts[5], nil
}

func WithVisibilityResetOnError(h HandlerFunc, client SQSClient, timeoutComputer VisibilityTimeoutComputer) HandlerFunc {
	return func(ctx context.Context, event events.SQSEvent) error {
		err := h(ctx, event)
		if err == nil {
			return nil
		}

		if len(event.Records) == 0 {
			return err
		}

		logger := zerolog.Ctx(ctx)

		accountID, queueName, parseErr := parseQueueARN(event.Records[0].EventSourceARN)
		if parseErr != nil {
			logger.Error().Err(parseErr).Msg("failed to parse queue ARN")
			return err
		}

		urlOutput, urlErr := client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
			QueueName:              &queueName,
			QueueOwnerAWSAccountId: &accountID,
		})
		if urlErr != nil {
			logger.Error().Err(urlErr).Msg("failed to get queue URL")
			return err
		}

		for _, msg := range event.Records {
			_, resetErr := client.ChangeMessageVisibility(ctx, &sqs.ChangeMessageVisibilityInput{
				QueueUrl:          urlOutput.QueueUrl,
				ReceiptHandle:     &msg.ReceiptHandle,
				VisibilityTimeout: timeoutComputer(msg),
			})
			if resetErr != nil {
				logger.Error().
					Err(resetErr).
					Str("messageId", msg.MessageId).
					Msg("failed to reset message visibility")
			} else {
				logger.Warn().
					Str("messageId", msg.MessageId).
					Msg("reset message visibility")
			}
		}

		return err
	}
}

func WithLogger(h HandlerFunc, logger zerolog.Logger) HandlerFunc {
	return func(ctx context.Context, event events.SQSEvent) error {
		return h(logger.WithContext(ctx), event)
	}
}

func WithXRayCapture(h HandlerFunc, segmentName string) HandlerFunc {
	return func(ctx context.Context, event events.SQSEvent) error {
		// For Lambda, we need to create a facade segment that references the Lambda-provided trace
		ctx, seg := xray.BeginFacadeSegment(ctx, segmentName, nil)
		defer seg.Close(nil)

		err := h(ctx, event)
		if err != nil {
			seg.AddError(err)
		}
		return err
	}
}
