package rabbitmqPool

import (
	"github.com/streadway/amqp"
)

/**
重试工具
*/
type retryClient struct {
	channel          *amqp.Channel
	data             *amqp.Delivery
	header           map[string]interface{}
	pool             *RabbitPool
	receive          *ConsumeReceive
}
type RetryClientInterface interface {
	Ack() error
}



func newRetryClient(channel *amqp.Channel, data *amqp.Delivery, header map[string]interface{}, pool *RabbitPool, receive *ConsumeReceive) *retryClient {
	return &retryClient{channel: channel, data: data, header: header, pool: pool, receive: receive}
}

func (r *retryClient) Ack() error {
	//如果是非自动确认消息 手动进行确认
	if ! r.receive.IsAutoAck {

		if r.data != nil {
			return r.data.Ack(true)
		}
		return ACK_DATA_NIL
	}
	return nil
}