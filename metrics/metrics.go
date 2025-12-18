package metrics

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

type CloudWatchClient interface {
	PutMetricData(ctx context.Context, params *cloudwatch.PutMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.PutMetricDataOutput, error)
}

type CloudWatchEmitter struct {
	client    CloudWatchClient
	namespace string
}

func NewCloudWatchEmitter(client CloudWatchClient, namespace string) *CloudWatchEmitter {
	return &CloudWatchEmitter{
		client:    client,
		namespace: namespace,
	}
}

func (e *CloudWatchEmitter) EmitGauge(ctx context.Context, name string, value float64) error {
	_, err := e.client.PutMetricData(ctx, &cloudwatch.PutMetricDataInput{
		Namespace: aws.String(e.namespace),
		MetricData: []types.MetricDatum{
			{
				MetricName: aws.String(name),
				Value:      aws.Float64(value),
				Unit:       types.StandardUnitCount,
			},
		},
	})
	return err
}
