package test

import (
	"fmt"
	"github.com/streadway/amqp"
	"github.com/insomniasary/rabbitmqPool"
	"sync"
	"testing"
)

var oncePool sync.Once
var instanceRPool *rabbitmqPool.RabbitPool

func TestProduct(t *testing.T) {
	initrabbitmq()
	run()
}

func initrabbitmq() *rabbitmqPool.RabbitPool {
	oncePool.Do(func() {
		instanceRPool = rabbitmqPool.NewProductPool()
		err := instanceRPool.Connect("127.0.0.1", 5672, "guest", "guest")
		if err != nil {
			fmt.Println(err)
		}
	})
	return instanceRPool
}

func run() {
	var wg sync.WaitGroup
	for i := 100; i < 200; i++ {
		wg.Add(1)
		go func(num int) {
			defer wg.Done()
			data := rabbitmqPool.GetRabbitMqDataFormat("", rabbitmqPool.EXCHANGE_TYPE_TOPIC, "testQueue1", "", fmt.Sprintf("这里是数据%d", num))
			channel, err := rabbitmqPool.GetProductChannel(instanceRPool)
			if err != nil {
				fmt.Println(err)
				return
			}
			_,err = channel.Ch.QueueDeclare(data.QueueName,false, false, false, false, nil)
			if err != nil {
				fmt.Println(err)
				return
			}
			publishing := amqp.Publishing{
				ContentType:  "text/plain",
				Body:         []byte(data.Data),
				DeliveryMode: amqp.Persistent, //持久化到磁盘
			}
			err = channel.Ch.Publish(data.ExchangeName,data.QueueName,false,false,publishing)
			if err != nil {
				fmt.Println(err)
				return
			}
		}(i)
	}
	wg.Wait()
}
