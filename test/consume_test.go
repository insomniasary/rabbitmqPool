package test

import (
	"fmt"
	"github.com/insomniasary/rabbitmqPool"
	"sync"
	"testing"
)

var onceConsumePool sync.Once
var instanceConsumePool *rabbitmqPool.RabbitPool
func TestConsume(t *testing.T)  {
	initConsumerabbitmq()
	Consume()
}
func initConsumerabbitmq() *rabbitmqPool.RabbitPool {
	onceConsumePool.Do(func() {
		instanceConsumePool = rabbitmqPool.NewConsumePool()
		err := instanceConsumePool.Connect("127.0.0.1", 5672, "guest", "guest")
		if err != nil {
			fmt.Println(err)
		}
	})
	return instanceConsumePool
}

func Consume() {
	nomrl := &rabbitmqPool.ConsumeReceive{
		ExchangeType: rabbitmqPool.EXCHANGE_TYPE_DIRECT,
		Route:        "",
		QueueName:    "testQueue1",
		IsAutoAck:    false, //自动消息确认
		EventFail: func(code int, e error, data []byte) {
			fmt.Printf("error:%s", e)
		},
		EventSuccess: func(data []byte, header map[string]interface{}, retryClient rabbitmqPool.RetryClientInterface) bool {
			err := retryClient.Ack()
			fmt.Println(string(data),err)
			return true
		},
	}
	instanceConsumePool.RegisterConsumeReceive(nomrl)
	err := instanceConsumePool.RunConsume()
	if err != nil {
		fmt.Println(err)
	}
}
