package rabbitmqPool

import (
	"fmt"
	"github.com/streadway/amqp"
	"sync"
	"sync/atomic"
	"time"
)

/**
消费者注册接收数据
*/
var statusLock sync.Mutex
var status bool = false

type ConsumeReceive struct {
	ExchangeName string                                                                                  //交换机
	ExchangeType string                                                                                  //交换机类型
	Route        string                                                                                  //路由
	QueueName    string                                                                                  //队列名称
	EventSuccess func(data []byte, header map[string]interface{}, retryClient RetryClientInterface) bool //成功事件回调
	EventFail    func(int, error, []byte)                                                                //失败回调

	IsAutoAck bool  //是否自动确认
}

/**
消费者处理
*/
func rConsume(pool *RabbitPool) {
	for _, v := range pool.consumeReceive {
		go func(pool *RabbitPool, receive *ConsumeReceive) {
			rListenerConsume(pool, receive)
		}(pool, v)
	}
	/**
	创建一个协程监听任务
	*/
	select {
	//case data := <-pool.errorChanel:
	case <-pool.errorChanel:
		statusLock.Lock()
		status = true
		statusLock.Unlock()
		retryConsume(pool)
	}
}

/**
监听消费
*/
func rListenerConsume(pool *RabbitPool, receive *ConsumeReceive) {
	var i int32 = 0
	for i = 0; i < pool.consumeMaxChannel; i++ {
		go func( p *RabbitPool, r *ConsumeReceive) {
			consumeTask(p, r)
		}(pool, receive)
	}
}
func retryConsume(pool *RabbitPool) {
	fmt.Printf("2秒后开始重试:[%d]\n", pool.consumeCurrentRetry)
	atomic.AddInt32(&pool.consumeCurrentRetry, 1)
	time.Sleep(time.Second * 2)
	_, err := rConnect(pool, true)
	if err != nil {
		retryConsume(pool)
	} else {
		statusLock.Lock()
		status = false
		statusLock.Unlock()
		_ = pool.initConnections(false)
		rConsume(pool)
	}
}

func consumeTask(pool *RabbitPool, receive *ConsumeReceive) {
	closeFlag := false
	pool.connectionLock.Lock()
	conn := pool.getConnection()
	if conn.conn.IsClosed() {
		var err error
		pool.connections[pool.clientType][pool.connectionIndex].conn, err = rConnect(pool, true)
		conn = pool.connections[pool.clientType][pool.connectionIndex]
		if err != nil {
			receive.EventFail(RCODE_CHANNEL_CREATE_ERROR, NewRabbitMqError(RCODE_CHANNEL_CREATE_ERROR, "conn create error", err.Error()), nil)
		}
	}
	pool.connectionLock.Unlock()
	//生成处理channel 根据最大channel数处理
	channel, err := rCreateChannel(conn)
	if err != nil {
		if receive.EventFail != nil {
			receive.EventFail(RCODE_CHANNEL_CREATE_ERROR, NewRabbitMqError(RCODE_CHANNEL_CREATE_ERROR, "channel create error", err.Error()), nil)
		}
		return
	}
	defer func() {
		_ = channel.Close()
		_ = conn.conn.Close()
	}()
	notifyClose := make(chan *amqp.Error)
	closeChan := make(chan *amqp.Error, 1)
	if err != nil {
		if receive.EventFail != nil {
			receive.EventFail(RCODE_CHANNEL_QUEUE_EXCHANGE_BIND_ERROR, NewRabbitMqError(RCODE_CHANNEL_QUEUE_EXCHANGE_BIND_ERROR, "交换机/队列/绑定失败", err.Error()), nil)
		}
		return
	}
	// 获取消费通道
	//确保rabbitmq会一个一个发消息
	_ = channel.Qos(1, 0, false)
	msgs, err := channel.Consume(
		receive.QueueName, // queue
		"",                // consumer
		false,             // auto-ack
		false,             // exclusive
		false,             // no-local
		false,             // no-wait
		nil,               // args
	)
	if nil != err {
		if receive.EventFail != nil {
			receive.EventFail(RCODE_GET_CHANNEL_ERROR, NewRabbitMqError(RCODE_GET_CHANNEL_ERROR, fmt.Sprintf("获取队列 %s 的消费通道失败", receive.QueueName), err.Error()), nil)
		}
		return
	}
	notifyClose = channel.NotifyClose(closeChan)
	for {
		select {
		case data := <-msgs:
			if receive.IsAutoAck { //如果是自动确认,否则需使用回调用 newRetryClient Ack
				_ = data.Ack(true)
			}
			if receive.EventSuccess != nil {
				retryClient := newRetryClient(channel, &data, data.Headers,pool, receive)
				receive.EventSuccess(data.Body, data.Headers, retryClient)
			}
		//一但有错误直接返回 并关闭信道
		case e := <-notifyClose:
			if receive.EventFail != nil {
				receive.EventFail(RCODE_CONNECTION_ERROR, NewRabbitMqError(RCODE_CONNECTION_ERROR, fmt.Sprintf("消息处理中断: queue:%s\n", receive.QueueName), e.Error()), nil)
			}
			setConnectError(pool, e.Code, fmt.Sprintf("消息处理中断: %s", e.Error()))
			closeFlag = true
		}
		if closeFlag {
			break
		}
	}
}
