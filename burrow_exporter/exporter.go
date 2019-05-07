package burrow_exporter

import (
	"context"

	"sync"
	"time"

	"net/http"

	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// BurrowExporter 结构体
type BurrowExporter struct {
	client                     *BurrowClient
	metricsListenAddr          string
	interval                   int
	wg                         sync.WaitGroup
	skipPartitionStatus        bool
	skipConsumerStatus         bool
	skipPartitionLag           bool
	skipPartitionCurrentOffset bool
	skipPartitionMaxOffset     bool
	skipTotalLag               bool
	skipTopicPartitionOffset   bool
}

// 处理group信息
func (be *BurrowExporter) processGroup(cluster, group string) {
	status, err := be.client.ConsumerGroupLag(cluster, group)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("error getting status for consumer group. returning.")
		return
	}

	// 设置prometheus指标，value类型都是 float64
	for _, partition := range status.Status.Partitions {
		if !be.skipPartitionLag {
			// 设置promethues分区消费堆积指标
			// eg: kafka_burrow_partition_lag{cluster="local",group="v-consumer",partition="0",topic="agora_data"} 0
			KafkaConsumerPartitionLag.With(prometheus.Labels{
				"cluster":   status.Status.Cluster,
				"group":     status.Status.Group,
				"topic":     partition.Topic,
				"partition": strconv.Itoa(int(partition.Partition)),
			}).Set(float64(partition.CurrentLag))
		}

		if !be.skipPartitionCurrentOffset {
			// 设置当前消费分区的偏移量指标
			// eg: kafka_burrow_partition_current_offset{cluster="local",group="v-consumer",partition="0",topic="agora_data"} 8941
			KafkaConsumerPartitionCurrentOffset.With(prometheus.Labels{
				"cluster":   status.Status.Cluster,
				"group":     status.Status.Group,
				"topic":     partition.Topic,
				"partition": strconv.Itoa(int(partition.Partition)),
			}).Set(float64(partition.End.Offset))
		}

		if !be.skipPartitionStatus {
			// 设置分区消费状态指标
			// eg: kafka_burrow_partition_status{cluster="local",group="v-consumer",partition="0",topic="agora_data"} 2
			// 可以看到 value是2， status:"OK" 在metrics.go map里表示 2
			KafkaConsumerPartitionCurrentStatus.With(prometheus.Labels{
				"cluster":   status.Status.Cluster,
				"group":     status.Status.Group,
				"topic":     partition.Topic,
				"partition": strconv.Itoa(int(partition.Partition)),
			}).Set(float64(Status[partition.Status]))
		}

		if !be.skipPartitionMaxOffset {
			// 设置分区消费最大偏移量指标
			// eg: kafka_burrow_partition_max_offset{cluster="local",group="v-consumer",partition="0",topic="agora_data"} 0
			// 每个分区的max_offset指标，默认0（其实是零值是0）
			KafkaConsumerPartitionMaxOffset.With(prometheus.Labels{
				"cluster":   status.Status.Cluster,
				"group":     status.Status.Group,
				"topic":     partition.Topic,
				"partition": strconv.Itoa(int(partition.Partition)),
			}).Set(float64(partition.End.MaxOffset))
		}
	}

	// 设置当前消费者组最大堆积指标
	// eg: kafka_burrow_total_lag{cluster="local",group="v-consumer"} 0
	if !be.skipTotalLag {
		KafkaConsumerTotalLag.With(prometheus.Labels{
			"cluster": status.Status.Cluster,
			"group":   status.Status.Group,
		}).Set(float64(status.Status.TotalLag))
	}

	if !be.skipConsumerStatus {
		KafkaConsumerStatus.With(prometheus.Labels{
			"cluster": status.Status.Cluster,
			"group":   status.Status.Group,
		}).Set(float64(Status[status.Status.Status]))
	}
}

// 处理每个topic的数据
func (be *BurrowExporter) processTopic(cluster, topic string) {
	details, err := be.client.ClusterTopicDetails(cluster, topic)
	if err != nil {
		log.WithFields(log.Fields{
			"err":   err,
			"topic": topic,
		}).Error("error getting status for cluster topic. returning.")
		return
	}

	if !be.skipTopicPartitionOffset {
		for i, offset := range details.Offsets {
			KafkaTopicPartitionOffset.With(prometheus.Labels{
				"cluster":   cluster,
				"topic":     topic,
				"partition": strconv.Itoa(i),
			}).Set(float64(offset))
		}
	}
}

// 抓取每个kafka集群的数据
func (be *BurrowExporter) processCluster(cluster string) {
	// 获取groups
	groups, err := be.client.ListConsumers(cluster)
	if err != nil {
		log.WithFields(log.Fields{
			"err":     err,
			"cluster": cluster,
		}).Error("error listing consumer groups. returning.")
		return
	}

	// 获取topics
	topics, err := be.client.ListClusterTopics(cluster)
	if err != nil {
		log.WithFields(log.Fields{
			"err":     err,
			"cluster": cluster,
		}).Error("error listing cluster topics. returning.")
		return
	}

	wg := sync.WaitGroup{}

	// 获取集群中每个group信息
	for _, group := range groups.ConsumerGroups {
		wg.Add(1)

		go func(g string) {
			defer wg.Done()
			be.processGroup(cluster, g)
		}(group)
	}

	for _, topic := range topics.Topics {
		wg.Add(1)

		go func(t string) {
			defer wg.Done()
			be.processTopic(cluster, t)
		}(topic)
	}

	wg.Wait()
}

// 对外暴露服务，供prometheus来 pull metrics
func (be *BurrowExporter) startPrometheus() {
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(be.metricsListenAddr, nil)
}

func (be *BurrowExporter) Close() {
	be.wg.Wait()
}

func (be *BurrowExporter) Start(ctx context.Context) {
	// 启动http服务, 对外暴露metrics
	be.startPrometheus()

	be.wg.Add(1)
	defer be.wg.Done()

	be.mainLoop(ctx)
}

// 抓取burrow
func (be *BurrowExporter) scrape() {
	start := time.Now()
	log.WithField("timestamp", start.UnixNano()).Info("Scraping burrow...")
	clusters, err := be.client.ListClusters()
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("error listing clusters. Continuing.")
		return
	}

	wg := sync.WaitGroup{}

	// 循环kafka集群，每个启动一个goroutine 抓取集群信息
	for _, cluster := range clusters.Clusters {
		wg.Add(1)

		go func(c string) {
			defer wg.Done()
			be.processCluster(c)
		}(cluster)
	}

	wg.Wait()

	end := time.Now()
	log.WithFields(log.Fields{
		"timestamp": end.UnixNano(),
		"took":      end.Sub(start),
	}).Info("Finished scraping burrow.")
}

// 主循环
// 起一个定时任务抓取burrow数据
func (be *BurrowExporter) mainLoop(ctx context.Context) {
	timer := time.NewTicker(time.Duration(be.interval) * time.Second)

	// scrape at app start without waiting for the first interval to elapse
	// 先行抓取
	be.scrape()

	for {
		select {
		case <-ctx.Done():
			log.Info("Shutting down exporter.")
			timer.Stop()
			return

		case <-timer.C:
			be.scrape()
		}
	}
}

func MakeBurrowExporter(burrowUrl string, apiVersion int, metricsAddr string, interval int, skipPartitionStatus bool,
	skipConsumerStatus bool, skipPartitionLag bool, skipPartitionCurrentOffset bool, skipPartitionMaxOffset bool, skipTotalLag bool, skipTopicPartitionOffset bool) *BurrowExporter {
	return &BurrowExporter{
		client:                     MakeBurrowClient(burrowUrl, apiVersion),
		metricsListenAddr:          metricsAddr,
		interval:                   interval,
		skipPartitionStatus:        skipPartitionStatus,
		skipConsumerStatus:         skipConsumerStatus,
		skipPartitionLag:           skipPartitionLag,
		skipPartitionCurrentOffset: skipPartitionCurrentOffset,
		skipPartitionMaxOffset:     skipPartitionMaxOffset,
		skipTotalLag:               skipTotalLag,
		skipTopicPartitionOffset:   skipTopicPartitionOffset,
	}
}
