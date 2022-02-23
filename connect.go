package rabbitmqPool

import (
	"errors"
	"fmt"
	"github.com/streadway/amqp"
)

func rConnect(r *RabbitPool, islock bool) (*amqp.Connection, error) {
	connectionUrl := fmt.Sprintf("amqp://%s:%s@%s:%d/", r.user, r.password, r.host, r.port)
	client, err := amqp.Dial(connectionUrl)
	if err != nil {
		return nil, err
	}
	return client, nil
}
/**
创建rabbitmq信道
*/
func rCreateChannel(conn *rConn) (*amqp.Channel, error) {
	ch, err := conn.conn.Channel()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Create Connect Channel Error: %s", err.Error()))
	}
	return ch, nil
}