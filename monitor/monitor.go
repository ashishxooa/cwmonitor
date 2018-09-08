package monitor

import (
	"time"

	"cwmonitor/metrics"
	"cwmonitor/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"
)

func GatherData(collectedMetrics []metrics.Metric) metrics.Data {
	data := metrics.Data{}
	for _, metric := range collectedMetrics {
		d, err := metric.Gather()
		if err != nil {
			log.Errorf("failed to Monitor data from metric [%s]: %s", metric.Name(), err)
			continue
		}

		data = append(data, d...)
	}

	return data
}

func convertPointToCloudWatch(point *metrics.Point) cloudwatch.MetricDatum {
	datum := cloudwatch.MetricDatum{
		MetricName: aws.String(point.Name),
		Unit:       aws.String(string(point.Unit)),
		Value:      aws.Float64(point.Value),
		Timestamp:  aws.Time(point.Timestamp),
		Dimensions: make([]*cloudwatch.Dimension, 0, len(point.Dimensions)),
	}

	for _, dim := range point.Dimensions {
		datum.Dimensions = append(datum.Dimensions, &cloudwatch.Dimension{
			Name: aws.String(dim.Name),
			Value: aws.String(dim.Value),
		})
	}
	return datum
}

func convertDataToCloudWatch(data metrics.Data) []*cloudwatch.MetricDatum {
	output := make([]*cloudwatch.MetricDatum, 0, len(data))
	for _, d := range data {
		datum := convertPointToCloudWatch(d)
		output = append(output, &datum)
	}
	return output
}

func PublishDataToCloudWatch(data metrics.Data, client cloudwatchiface.CloudWatchAPI, namespace string) error {
	multierror := util.MultiError{}
	for _, d := range data.Batch(20) {
		datum := convertDataToCloudWatch(d)
		_, err := client.PutMetricData(&cloudwatch.PutMetricDataInput{
			Namespace: aws.String(namespace),
			MetricData: datum,
		})
		if err != nil {
			multierror.Add(err)
		}
	}

	return errors.Wrap(multierror.ErrorOrNil(), "failed to put metric data")
}

func Monitor(metrics []metrics.Metric, extraDimensions []metrics.Dimension, client cloudwatchiface.CloudWatchAPI, namespace string) {
	data := GatherData(metrics)
	data.AddDimensions(extraDimensions...)
	err := PublishDataToCloudWatch(data, client, namespace)
	if err != nil {
		log.Errorf("failed to PublishDataToCloudWatch metrics: %s", err)
	}
}

func Exec(c Config) error {
	log.Infof("cwmonitor - version: %s", c.Version)

	err := c.Validate()
	if err != nil {
		return errors.Wrap(err, "invalid inputs")
	}

	sess := session.Must(session.NewSession())
	svc := cloudwatch.New(sess)

	if c.Once {
		Monitor(c.GetRequestedMetrics(), c.GetExtraDimensions(), svc, c.Namespace)
	} else {
		log.Infof("interval: %d minutes", c.Interval)
		for range time.Tick(c.Interval) {
			Monitor(c.GetRequestedMetrics(), c.GetExtraDimensions(), svc, c.Namespace)
		}
	}

	return nil
}