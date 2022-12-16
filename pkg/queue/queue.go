package queue

import (
	"go-sniffer/pkg/model"
	"k8s.io/klog/v2"
	"time"
)

var Metrics *MetricsQueue

func init() {
	Metrics = &MetricsQueue{
		LocalEventsBuffer: make(chan *model.MysqlQueryPiece, LocalEventsBufferSize),
	}
}

const (
	LocalEventsBufferSize   = 10000
	DefaultPushLogFrequency = 5 * time.Second
	DefaultThreadPoolSize   = 10
	DefaultRetry            = 3
)

type MetricsQueue struct {
	LocalEventsBuffer chan *model.MysqlQueryPiece
	Host              string
}

type MetricsBatch struct {
	Timestamp time.Time
	Records   []*model.MysqlQueryPiece
}

func (k *MetricsQueue) Add(record *model.MysqlQueryPiece) {
	if record != nil {
		select {
		case k.LocalEventsBuffer <- record:
			// Ok, buffer not full.
		default:
			// Buffer full, need to drop the event.
			klog.Errorf("alert event buffer full, dropping event")
		}
	}
}

func (k *MetricsQueue) Export() {
	for {
		now := time.Now()
		start := now.Truncate(DefaultPushLogFrequency)
		end := start.Add(DefaultPushLogFrequency)
		timeToNextSync := end.Sub(now)

		select {
		case <-time.After(timeToNextSync):
			alerts := k.GetNewMetrics()
			if len(alerts.Records) > 0 {
				k.ExportEvents(alerts)
			}
		}
	}
}

func (k *MetricsQueue) GetNewMetrics() *MetricsBatch {
	result := &MetricsBatch{
		Timestamp: time.Now(),
		Records:   []*model.MysqlQueryPiece{},
	}
logLoop:
	for {
		select {
		case event := <-k.LocalEventsBuffer:
			result.Records = append(result.Records, event)
		default:
			break logLoop
		}
	}

	return result
}

func (k *MetricsQueue) ExportEvents(alertBatch *MetricsBatch) {
	k.PushMetricsController(alertBatch)
}

func (k *MetricsQueue) PushMetricsController(alertBatch *MetricsBatch) {
	var thread int
	thread = DefaultThreadPoolSize
	chJobs := make(chan *model.MysqlQueryPiece, len(alertBatch.Records))

	for w := 1; w <= thread; w++ {
		go k.PushWork(chJobs)
	}

	for _, record := range alertBatch.Records {
		chJobs <- record
	}
	close(chJobs)

}

func (k *MetricsQueue) PushWork(jobs <-chan *model.MysqlQueryPiece) {
	for j := range jobs {
		rt := DefaultRetry
		for {
			err := k.push(j)
			if err != nil {
				klog.Errorf("fail to push traffic inject alert,retry: %d, err: %v", rt, err)
				rt--
				if rt == 0 {
					break
				}
				continue
			}
			break
		}
	}
}

func (k *MetricsQueue) push(record *model.MysqlQueryPiece) error {
	return k.PushMetrics(record)
}
