package sqs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestWithReducedContextDeadline(t *testing.T) {
	testCases := []struct {
		name              string
		setupContext      func() (context.Context, context.CancelFunc)
		buffer            time.Duration
		handlerErr        error
		expectErr         bool
		expectErrContains string
		expectHandlerCall bool
		validateDeadline  func(t *testing.T, ctx context.Context, originalDeadline time.Time)
	}{
		{
			name: "reduces deadline by buffer",
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(context.Background(), time.Now().Add(30*time.Second))
			},
			buffer:            5 * time.Second,
			expectHandlerCall: true,
			validateDeadline: func(t *testing.T, ctx context.Context, originalDeadline time.Time) {
				deadline, ok := ctx.Deadline()
				require.True(t, ok)
				expected := originalDeadline.Add(-5 * time.Second)
				assert.WithinDuration(t, expected, deadline, time.Millisecond)
			},
		},
		{
			name: "fails fast when buffer exceeds remaining time",
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
			},
			buffer:            10 * time.Second,
			expectErr:         true,
			expectErrContains: "attempt to reduce deadline by more than possible",
			expectHandlerCall: false,
		},
		{
			name: "passes through when no deadline present",
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			buffer:            5 * time.Second,
			expectHandlerCall: true,
			validateDeadline: func(t *testing.T, ctx context.Context, _ time.Time) {
				_, ok := ctx.Deadline()
				assert.False(t, ok, "should not have a deadline")
			},
		},
		{
			name: "propagates handler error",
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(context.Background(), time.Now().Add(30*time.Second))
			},
			buffer:            5 * time.Second,
			handlerErr:        errors.New("handler failed"),
			expectErr:         true,
			expectErrContains: "handler failed",
			expectHandlerCall: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := tc.setupContext()
			defer cancel()

			originalDeadline, _ := ctx.Deadline()

			handlerCalled := false
			var capturedCtx context.Context
			handler := func(ctx context.Context, event events.SQSEvent) error {
				handlerCalled = true
				capturedCtx = ctx
				return tc.handlerErr
			}

			wrapped := WithReducedContextDeadline(handler, tc.buffer)
			err := wrapped(ctx, events.SQSEvent{})

			assert.Equal(t, tc.expectHandlerCall, handlerCalled)

			if tc.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrContains)
			} else {
				require.NoError(t, err)
			}

			if tc.validateDeadline != nil && handlerCalled {
				tc.validateDeadline(t, capturedCtx, originalDeadline)
			}
		})
	}
}

func TestWithPanicProtection(t *testing.T) {
	originalErr := errors.New("original error")

	testCases := []struct {
		name              string
		handler           func(ctx context.Context, event events.SQSEvent) error
		expectErr         bool
		expectErrContains string
		expectWrappedErr  error
	}{
		{
			name: "normal execution passes through",
			handler: func(ctx context.Context, event events.SQSEvent) error {
				return nil
			},
			expectErr: false,
		},
		{
			name: "propagates handler error",
			handler: func(ctx context.Context, event events.SQSEvent) error {
				return errors.New("handler error")
			},
			expectErr:         true,
			expectErrContains: "handler error",
		},
		{
			name: "recovers from string panic",
			handler: func(ctx context.Context, event events.SQSEvent) error {
				panic("something went wrong")
			},
			expectErr:         true,
			expectErrContains: "recovered from panic: something went wrong",
		},
		{
			name: "recovers from error panic and preserves wrapping",
			handler: func(ctx context.Context, event events.SQSEvent) error {
				panic(originalErr)
			},
			expectErr:         true,
			expectErrContains: "recovered from panic",
			expectWrappedErr:  originalErr,
		},
		{
			name: "recovers from other panic types",
			handler: func(ctx context.Context, event events.SQSEvent) error {
				panic(42)
			},
			expectErr:         true,
			expectErrContains: "recovered from panic: 42",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wrapped := WithPanicProtection(tc.handler)
			err := wrapped(context.Background(), events.SQSEvent{})

			if tc.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrContains)

				if tc.expectWrappedErr != nil {
					assert.True(t, errors.Is(err, tc.expectWrappedErr), "should preserve error chain")
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLinearVisibilityTimeoutComputer(t *testing.T) {
	testCases := []struct {
		name          string
		step          time.Duration
		receiveCount  string
		expectTimeout int32
	}{
		{
			name:          "first receive returns zero",
			step:          30 * time.Second,
			receiveCount:  "1",
			expectTimeout: 0,
		},
		{
			name:          "second receive returns one step",
			step:          30 * time.Second,
			receiveCount:  "2",
			expectTimeout: 30,
		},
		{
			name:          "third receive returns two steps",
			step:          30 * time.Second,
			receiveCount:  "3",
			expectTimeout: 60,
		},
		{
			name:          "missing attribute defaults to first receive",
			step:          30 * time.Second,
			receiveCount:  "",
			expectTimeout: 0,
		},
		{
			name:          "malformed attribute defaults to first receive",
			step:          30 * time.Second,
			receiveCount:  "not-a-number",
			expectTimeout: 0,
		},
		{
			name:          "different step size",
			step:          60 * time.Second,
			receiveCount:  "4",
			expectTimeout: 180,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			computer := LinearVisibilityTimeoutComputer(tc.step)

			msg := events.SQSMessage{}
			if tc.receiveCount != "" {
				msg.Attributes = map[string]string{
					"ApproximateReceiveCount": tc.receiveCount,
				}
			}

			timeout := computer(msg)
			assert.Equal(t, tc.expectTimeout, timeout)
		})
	}
}

func TestParseQueueARN(t *testing.T) {
	testCases := []struct {
		name            string
		arn             string
		expectAccountID string
		expectQueueName string
		expectErr       bool
	}{
		{
			name:            "valid ARN",
			arn:             "arn:aws:sqs:us-east-1:123456789012:my-queue",
			expectAccountID: "123456789012",
			expectQueueName: "my-queue",
			expectErr:       false,
		},
		{
			name:      "invalid ARN - too few parts",
			arn:       "arn:aws:sqs:us-east-1",
			expectErr: true,
		},
		{
			name:      "invalid ARN - too many parts",
			arn:       "arn:aws:sqs:us-east-1:123:queue:extra",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			accountID, queueName, err := parseQueueARN(tc.arn)

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectAccountID, accountID)
				assert.Equal(t, tc.expectQueueName, queueName)
			}
		})
	}
}

func TestWithVisibilityResetOnError(t *testing.T) {
	type getQueueUrlCall struct {
		queueName string
		accountID string
		result    string
		err       error
	}

	type changeVisibilityCall struct {
		queueURL          string
		receiptHandle     string
		visibilityTimeout int32
		err               error
	}

	queueARN := "arn:aws:sqs:us-east-1:123456789012:test-queue"
	queueURL := "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue"

	testCases := []struct {
		name                  string
		messages              []events.SQSMessage
		handlerErr            error
		getQueueUrlCall       *getQueueUrlCall
		changeVisibilityCalls []changeVisibilityCall
		expectErr             bool
		expectErrContains     string
	}{
		{
			name: "no error skips visibility reset",
			messages: []events.SQSMessage{
				{MessageId: "msg-1", ReceiptHandle: "handle-1", EventSourceARN: queueARN},
			},
			handlerErr: nil,
			expectErr:  false,
		},
		{
			name: "error triggers visibility reset for single message",
			messages: []events.SQSMessage{
				{
					MessageId:      "msg-1",
					ReceiptHandle:  "handle-1",
					EventSourceARN: queueARN,
					Attributes:     map[string]string{"ApproximateReceiveCount": "1"},
				},
			},
			handlerErr: errors.New("processing failed"),
			getQueueUrlCall: &getQueueUrlCall{
				queueName: "test-queue",
				accountID: "123456789012",
				result:    queueURL,
			},
			changeVisibilityCalls: []changeVisibilityCall{
				{queueURL: queueURL, receiptHandle: "handle-1", visibilityTimeout: 0},
			},
			expectErr:         true,
			expectErrContains: "processing failed",
		},
		{
			name: "error triggers visibility reset for multiple messages",
			messages: []events.SQSMessage{
				{
					MessageId:      "msg-1",
					ReceiptHandle:  "handle-1",
					EventSourceARN: queueARN,
					Attributes:     map[string]string{"ApproximateReceiveCount": "2"},
				},
				{
					MessageId:      "msg-2",
					ReceiptHandle:  "handle-2",
					EventSourceARN: queueARN,
					Attributes:     map[string]string{"ApproximateReceiveCount": "3"},
				},
			},
			handlerErr: errors.New("batch failed"),
			getQueueUrlCall: &getQueueUrlCall{
				queueName: "test-queue",
				accountID: "123456789012",
				result:    queueURL,
			},
			changeVisibilityCalls: []changeVisibilityCall{
				{queueURL: queueURL, receiptHandle: "handle-1", visibilityTimeout: 30},
				{queueURL: queueURL, receiptHandle: "handle-2", visibilityTimeout: 60},
			},
			expectErr:         true,
			expectErrContains: "batch failed",
		},
		{
			name: "GetQueueUrl failure logs error and returns original error",
			messages: []events.SQSMessage{
				{
					MessageId:      "msg-1",
					ReceiptHandle:  "handle-1",
					EventSourceARN: queueARN,
					Attributes:     map[string]string{"ApproximateReceiveCount": "1"},
				},
			},
			handlerErr: errors.New("processing failed"),
			getQueueUrlCall: &getQueueUrlCall{
				queueName: "test-queue",
				accountID: "123456789012",
				err:       errors.New("SQS GetQueueUrl error"),
			},
			expectErr:         true,
			expectErrContains: "processing failed",
		},
		{
			name: "visibility reset failure is logged but original error returned",
			messages: []events.SQSMessage{
				{
					MessageId:      "msg-1",
					ReceiptHandle:  "handle-1",
					EventSourceARN: queueARN,
					Attributes:     map[string]string{"ApproximateReceiveCount": "1"},
				},
			},
			handlerErr: errors.New("processing failed"),
			getQueueUrlCall: &getQueueUrlCall{
				queueName: "test-queue",
				accountID: "123456789012",
				result:    queueURL,
			},
			changeVisibilityCalls: []changeVisibilityCall{
				{queueURL: queueURL, receiptHandle: "handle-1", visibilityTimeout: 0, err: errors.New("SQS error")},
			},
			expectErr:         true,
			expectErrContains: "processing failed",
		},
	}

	timeoutComputer := LinearVisibilityTimeoutComputer(30 * time.Second)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := NewMockSQSClient(t)

			if tc.getQueueUrlCall != nil {
				var result *sqs.GetQueueUrlOutput
				if tc.getQueueUrlCall.result != "" {
					result = &sqs.GetQueueUrlOutput{QueueUrl: &tc.getQueueUrlCall.result}
				}
				mockClient.EXPECT().
					GetQueueUrl(mock.Anything, &sqs.GetQueueUrlInput{
						QueueName:              &tc.getQueueUrlCall.queueName,
						QueueOwnerAWSAccountId: &tc.getQueueUrlCall.accountID,
					}).
					Return(result, tc.getQueueUrlCall.err)
			}

			for _, call := range tc.changeVisibilityCalls {
				mockClient.EXPECT().
					ChangeMessageVisibility(mock.Anything, &sqs.ChangeMessageVisibilityInput{
						QueueUrl:          &call.queueURL,
						ReceiptHandle:     &call.receiptHandle,
						VisibilityTimeout: call.visibilityTimeout,
					}).
					Return(&sqs.ChangeMessageVisibilityOutput{}, call.err)
			}

			handler := func(ctx context.Context, event events.SQSEvent) error {
				return tc.handlerErr
			}

			wrapped := WithVisibilityResetOnError(handler, mockClient, timeoutComputer)
			err := wrapped(context.Background(), events.SQSEvent{Records: tc.messages})

			if tc.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}