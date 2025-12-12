package event

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type SQSClient interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

type SQSEventDispatcher struct {
	client   SQSClient
	queueURL string
}

func NewSQSEventDispatcher(client SQSClient, queueURL string) *SQSEventDispatcher {
	return &SQSEventDispatcher{
		client:   client,
		queueURL: queueURL,
	}
}

func (d *SQSEventDispatcher) PublishEvent(ctx context.Context, event any) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = d.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(d.queueURL),
		MessageBody: aws.String(string(body)),
	})
	return err
}
