package burrow_exporter

import "github.com/prometheus/client_golang/prometheus"

// If we are missing a status, it will return 0
// NOTFOUND – 消费组未找到
// OK – 消费组状态正常
// WARN – 消费组处在WARN状态，例如offset在移动但是Lag不停增长。 the offsets are moving but lag is increasing
// ERR – 消费组处在ERR状态。例如，offset停止变动，但Lag非零。 the offsets have stopped for one or more partitions but lag is non-zero
// STOP – 消费组处在ERR状态。例如offset长时间未提交。the offsets have not been committed in a log period of time
// STALL – 消费组处在STALL状态。例如offset已提交但是没有变化，Lag非零。the offsets are being committed, but they are not changing and the lag is non-zero
var Status = map[string]int{
	"NOTFOUND": 1,
	"OK":       2,
	"WARN":     3,
	"ERR":      4,
	"STOP":     5,
	"STALL":    6,
	"REWIND":   7,
}

var (
	// 下面指标都是 Gauge类型
	// 分区消费堆积指标
	KafkaConsumerPartitionLag = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_burrow_partition_lag",
			Help: "The lag of the latest offset commit on a partition as reported by burrow.",
		},
		// labels
		[]string{"cluster", "group", "topic", "partition"},
	)

	// 当前消费分区的偏移量
	KafkaConsumerPartitionCurrentOffset = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_burrow_partition_current_offset",
			Help: "The latest offset commit on a partition as reported by burrow.",
		},
		[]string{"cluster", "group", "topic", "partition"},
	)
	// 分区消费状态
	KafkaConsumerPartitionCurrentStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_burrow_partition_status",
			Help: "The status of a partition as reported by burrow.",
		},
		[]string{"cluster", "group", "topic", "partition"},
	)
	// 分区消费最大偏移量
	KafkaConsumerPartitionMaxOffset = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_burrow_partition_max_offset",
			Help: "The log end offset on a partition as reported by burrow.",
		},
		[]string{"cluster", "group", "topic", "partition"},
	)
	// 当前消费者组最大堆积
	KafkaConsumerTotalLag = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_burrow_total_lag",
			Help: "The total amount of lag for the consumer group as reported by burrow",
		},
		[]string{"cluster", "group"},
	)
	// 消费状态
	KafkaConsumerStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_burrow_status",
			Help: "The status of a partition as reported by burrow.",
		},
		[]string{"cluster", "group"},
	)
	// topic 分区 offset
	KafkaTopicPartitionOffset = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_burrow_topic_partition_offset",
			Help: "The latest offset on a topic's partition as reported by burrow.",
		},
		[]string{"cluster", "topic", "partition"},
	)
)

// 初始化prometheus metrics 注册
func init() {
	prometheus.MustRegister(KafkaConsumerPartitionLag)
	prometheus.MustRegister(KafkaConsumerPartitionCurrentOffset)
	prometheus.MustRegister(KafkaConsumerPartitionCurrentStatus)
	prometheus.MustRegister(KafkaConsumerPartitionMaxOffset)
	prometheus.MustRegister(KafkaConsumerTotalLag)
	prometheus.MustRegister(KafkaConsumerStatus)
	prometheus.MustRegister(KafkaTopicPartitionOffset)
}
