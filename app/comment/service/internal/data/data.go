package data

import (
	"Atreus/app/comment/service/internal/conf"
	"Atreus/app/comment/service/pkg/gormX"
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis/v8"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"sync"
)

var ProviderSet = wire.NewSet(NewData, NewCommentRepo, NewUserRepo, NewPublishRepo, NewMysqlConn, NewRedisConn)

//type message struct {
//	writer *kafka.Writer
//	reader *kafka.Reader
//}

type Data struct {
	db    *gormX.DB
	cache *redis.Client
	//messageQueue *message
	log *log.Helper
}

func NewData(db *gorm.DB, cacheClient *redis.Client, logger log.Logger) (*Data, func(), error) {
	logHelper := log.NewHelper(log.With(logger, "module", "data/comment"))
	// 并发关闭所有数据库连接，后期根据Redis与Mysql是否数据同步修改
	cleanup := func() {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := cacheClient.Ping(context.Background()).Result()
			if err != nil {
				logHelper.Warn("Redis connection pool is empty")
				return
			}
			if err = cacheClient.Close(); err != nil {
				logHelper.Errorf("Redis connection closure failed, err: %w", err)
			}
			logHelper.Info("Successfully close the Redis connection")
		}()
		//wg.Add(1)
		//go func() {
		//	defer wg.Done()
		//	err := messageQueue.writer.Close()
		//	if err != nil {
		//		logHelper.Errorf("Kafka connection closure failed, err: %w", err)
		//		return
		//	}
		//	err = messageQueue.reader.Close()
		//	if err != nil {
		//		logHelper.Errorf("Kafka connection closure failed, err: %w", err)
		//		return
		//	}
		//	logHelper.Info("Successfully close the Kafka connection")
		//}()
		wg.Wait()
	}

	data := &Data{
		db:    gormX.NewConn(db),
		cache: cacheClient,
		//messageQueue: messageQueue,
		log: logHelper,
	}
	return data, cleanup, nil
}

// NewMysqlConn mysql数据库连接
func NewMysqlConn(c *conf.Data) *gorm.DB {
	db, err := gorm.Open(mysql.Open(c.Mysql.Dsn))
	if err != nil {
		log.Fatalf("Database connection failure, err : %v", err)
	}
	InitDB(db)
	log.Info("Database enabled successfully!")
	return db.Model(&Comment{})
}

// NewRedisConn Redis数据库连接
func NewRedisConn(c *conf.Data) *redis.Client {
	client := redis.NewClient(&redis.Options{
		DB:           int(c.Redis.CommentDb),
		Addr:         c.Redis.Addr,
		WriteTimeout: c.Redis.WriteTimeout.AsDuration(),
		ReadTimeout:  c.Redis.ReadTimeout.AsDuration(),
		Password:     c.Redis.Password,
	})

	// ping Redis客户端，判断连接是否存在
	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("Redis database connection failure, err : %v", err)
	}
	log.Info("Cache enabled successfully!")
	return client
}

//func NewKafkaConn(c *conf.Data) *message {
//	// 初始化kafka写入器
//	writer := &kafka.Writer{
//		Addr:                   kafka.TCP(c.Kafka.Addr),
//		Topic:                  c.Kafka.Topic,
//		WriteTimeout:           c.Kafka.WriteTimeout.AsDuration(),
//		Balancer:               &kafka.Hash{},
//		RequiredAcks:           kafka.RequireAll,
//		AllowAutoTopicCreation: true,
//	}
//	// 初始化kafka读取器
//	reader := kafka.NewReader(kafka.ReaderConfig{
//		Brokers:        []string{c.Kafka.Addr},
//		Topic:          c.Kafka.Topic,
//		MaxAttempts:    3,
//		CommitInterval: 1 * time.Second,
//		StartOffset:    kafka.FirstOffset,
//	})
//	messageQueue := &message{
//		writer: writer,
//		reader: reader,
//	}
//	log.Info("MessageQueue enabled successfully!")
//	return messageQueue
//}

// InitDB 创建Comments数据表，并自动迁移
func InitDB(db *gorm.DB) {
	if err := db.AutoMigrate(&Comment{}); err != nil {
		log.Fatalf("Database initialization error, err : %v", err)
	}
}
