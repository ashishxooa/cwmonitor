package monitor

import (
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/dedalusj/cwmonitor/metrics"
	"github.com/dedalusj/cwmonitor/util"
	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"
)

type Config struct {
	Namespace string
	Interval  time.Duration
	HostId    string
	Metrics   string
	Once      bool
	Metadata  util.Metadata
	Client    cloudwatchiface.CloudWatchAPI
}

func (c Config) Validate() error {
	err := util.MultiError{}

	if c.Namespace == "" {
		err.Add(errors.New("namespace cannot be empty"))
	}
	if c.Interval == time.Duration(0) {
		err.Add(errors.New("interval cannot be zero"))
	}
	if c.HostId == "" {
		err.Add(errors.New("hostid cannot be empty"))
	}
	if c.Metrics == "" {
		err.Add(errors.New("metrics cannot be empty"))
	}

	return err.ErrorOrNil()
}

func (c Config) GetRequestedMetrics() []metrics.Metric {
	metricsSet := map[string]bool{}
	for _, m := range strings.Split(c.Metrics, ",") {
		metricsSet[m] = true
	}

	collectedMetrics := make([]metrics.Metric, 0, len(metricsSet))
	for m := range metricsSet {
		switch m {
		case "memory":
			collectedMetrics = append(collectedMetrics, metrics.Memory{})
		case "swap":
			collectedMetrics = append(collectedMetrics, metrics.Swap{})
		case "disk":
			collectedMetrics = append(collectedMetrics, metrics.Disk{})
		case "cpu":
			collectedMetrics = append(collectedMetrics, metrics.CPU{})
		case "docker-stats":
			collectedMetrics = append(collectedMetrics, metrics.DockerStat{})
		case "docker-health":
			collectedMetrics = append(collectedMetrics, metrics.DockerHealth{})
		case "":
			continue
		default:
			log.Warnf("unknown metric: %s", m)
		}
	}

	return collectedMetrics
}

func (c Config) GetExtraDimensions() []metrics.Dimension {
	extraDimensions, _ := metrics.MapToDimensions(map[string]string{"Host": c.HostId})
	return extraDimensions
}

func (c Config) GetTicker() *time.Ticker {
	return time.NewTicker(c.Interval)
}
