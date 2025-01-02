package article

import (
	"context"
	"github.com/IBM/sarama"
	"time"
	"webook/internal/repository"
	"webook/pkg/logger"
	"webook/pkg/saramax"
)

// InteractiveReadEventConsumer 定义了一个消费 Kafka 消息的消费者
// 用于消费来自 topicReadEvent topic的文章阅读事件
// 该消费者会处理每个事件并根据事件信息更新文章的阅读计数
type InteractiveReadEventConsumer struct {
	client sarama.Client // Kafka 客户端，用于连接 Kafka 集群
	repo   repository.InteractiveRepository
	l      logger.Logger
}

func NewInteractiveReadEventConsumer(client sarama.Client, l logger.Logger, repo repository.InteractiveRepository) *InteractiveReadEventConsumer {
	return &InteractiveReadEventConsumer{
		client: client,
		l:      l,
		repo:   repo,
	}
}

// Start 启动消费者组并开始消费来自 topicReadEvent topic的消息
// 该方法会创建一个 Kafka 消费者组，异步启动消息消费过程
func (r *InteractiveReadEventConsumer) Start() error {
	// 使用 Kafka 客户端创建消费者组，"interactive" 是消费者组的名称
	cg, err := sarama.NewConsumerGroupFromClient("interactive", r.client)
	if err != nil {
		return err
	}

	// 异步启动消息消费过程，使用 topicReadEvent topic
	go func() {
		err := cg.Consume(context.Background(), []string{topicReadEvent},
			saramax.NewHandler[ReadEvent](r.l, r.Consume))
		if err != nil {
			// 如果消费过程中发生错误，记录错误日志
			r.l.Error("退出了消费循环异常", logger.Error(err))
		}
	}()
	return err
}

// Consume 消费者处理每条消息的方法，处理文章阅读事件并更新相应文章的阅读计数
//
//	msg: Kafka 消息，包含了从 Kafka topic接收到的消息内容
//	t: 消息内容的解码后的结构体，这里是 ReadEvent 对象，包含用户 ID 和文章 ID
func (r *InteractiveReadEventConsumer) Consume(msg *sarama.ConsumerMessage, t ReadEvent) error {
	// 创建一个带有超时限制的上下文，用于数据库操作
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel() // 确保在函数结束时取消上下文
	// 调用仓库方法更新文章的阅读计数
	return r.repo.IncrReadCnt(ctx, "article", t.Aid)
}

// InteractiveReadEventBatchConsumer 批量消费 Kafka 中的阅读事件消息
// 该消费者从 Kafka 中读取指定 topic 的消息
// 并在消费过程中批量处理这些事件，更新数据库中的阅读计数
type InteractiveReadEventBatchConsumer struct {
	client sarama.Client                    // Kafka 客户端，用于连接 Kafka 集群
	repo   repository.InteractiveRepository // 用于操作数据库的 repository，更新阅读计数
	l      logger.Logger                    // 日志记录器，用于记录日志
}

func NewInteractiveReadEventBatchConsumer(client sarama.Client, l logger.Logger, repo repository.InteractiveRepository) *InteractiveReadEventBatchConsumer {
	return &InteractiveReadEventBatchConsumer{
		client: client,
		l:      l,
		repo:   repo,
	}
}

// Start 启动消费者，开始消费 Kafka 中的消息。
// 该方法会创建一个消费者组并开始从指定的 Kafka topic 中读取消息。
func (r *InteractiveReadEventBatchConsumer) Start() error {
	// 使用 Kafka 客户端创建消费者组，"interactive" 是消费者组的名称
	cg, err := sarama.NewConsumerGroupFromClient("interactive", r.client)
	if err != nil {
		// 如果创建消费者组失败，返回错误
		return err
	}

	// 异步启动消息消费过程，消费来自 topicReadEvent 的消息
	go func() {
		err := cg.Consume(context.Background(), []string{topicReadEvent},
			saramax.NewBatchHandler[ReadEvent](r.l, r.Consume))
		if err != nil {
			// 如果消费过程中发生错误，记录错误日志并退出
			r.l.Error("退出了消费循环异常", logger.Error(err))
		}
	}()

	return err
}

// Consume 批量消费 Kafka 消息，将多个阅读事件一起处理，更新多个文章的阅读计数
//
//	msgs: Kafka 消费的消息列表，每条消息对应一个阅读事件
//	evts: 解析后的阅读事件列表，每个事件包含用户 ID 和文章 ID
func (r *InteractiveReadEventBatchConsumer) Consume(msgs []*sarama.ConsumerMessage, evts []ReadEvent) error {
	// 创建一个带有超时限制的上下文，1秒超时
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel() // 确保在函数结束时取消上下文

	// 如果没有消息，从这儿就返回
	if len(msgs) == 0 {
		return nil
	}

	// 创建两个切片：bizs 用于存储业务标识（这里是 "article"），
	// ids 用于存储每个事件中的用户 ID（这里是文章的 UID）
	bizs := make([]string, 0, len(msgs)) // 初始化 bizs 切片
	ids := make([]int64, 0, len(msgs))   // 初始化 ids 切片

	// 遍历 evts 列表（每个 ReadEvent 对象），提取对应的业务标识（"article"）和用户 ID
	for _, evt := range evts {
		bizs = append(bizs, "article") // 所有事件的业务标识都为 "article"
		ids = append(ids, evt.Aid)     // 将每个事件的文章 ID 加入 ids 列表
	}
	// 调用仓库的 BatchIncrReadCnt 方法，批量更新多个文章的阅读计数
	return r.repo.BatchIncrReadCnt(ctx, bizs, ids)
}
