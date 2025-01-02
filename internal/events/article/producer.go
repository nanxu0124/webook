package article

import (
	"context"
	"encoding/json"
	"github.com/IBM/sarama"
)

// topicReadEvent 定义了 Kafka 消息队列中的topic，用于接收文章阅读事件
// 在这里，所有与文章阅读相关的事件都会被发送到该topic
const topicReadEvent = "article_read_event"

// Producer 生产者接口
type Producer interface {
	// ProduceReadEvent 用于发送文章阅读事件
	ProduceReadEvent(ctx context.Context, evt ReadEvent) error
}

// KafkaProducer 定义了 Kafka 消息生产者的实现，它实现了 Producer 接口
type KafkaProducer struct {
	producer sarama.SyncProducer // 使用 sarama.SyncProducer 进行消息发送
}

func NewKafkaProducer(pc sarama.SyncProducer) Producer {
	return &KafkaProducer{
		producer: pc,
	}
}

// ProduceReadEvent 将读取事件（ReadEvent）转换为 JSON 格式，并发送到 Kafka topic中
func (k *KafkaProducer) ProduceReadEvent(ctx context.Context, evt ReadEvent) error {
	// 将事件对象（ReadEvent）转换为 JSON 格式
	data, err := json.Marshal(evt)
	if err != nil {
		return err // 如果转换失败，返回错误
	}

	// 创建一个 Kafka 消息对象，设置消息的topic和消息体（value）
	_, _, err = k.producer.SendMessage(&sarama.ProducerMessage{
		Topic: topicReadEvent,           // 设置消息的topic为 "article_read_event"
		Value: sarama.ByteEncoder(data), // 消息体为 JSON 编码后的事件数据
	})

	return err // 返回发送消息时的错误（如果有的话）
}

// ReadEvent 定义了一个文章阅读事件的结构体
// 包含了用户 ID（Uid）和文章 ID（Aid），表示某个用户阅读了某篇文章
type ReadEvent struct {
	Uid int64 // 用户 ID，标识阅读文章的用户
	Aid int64 // 文章 ID，标识被阅读的文章
}
