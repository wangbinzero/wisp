package middleware

import (
	"github.com/streadway/amqp"
	"gopkg.in/yaml.v2"
	"io"
	"wisp/log"
)

type (
	Exchange struct {
		Durable    bool      `yaml:"durable"`
		AutoDelete bool      `yaml:"autodelete"`
		Internal   bool      `yaml:"internal"`
		Nowait     bool      `yaml:"nowait"`
		Type       string    `yaml:"type"`
		Bindings   []Binding `yaml:"bindings"`
	}

	Binding struct {
		Exchange string `yaml:"exchange"`
		Key      string `yaml:"key"`
		Nowait   bool   `yaml:"nowait"`
	}

	QueueSpec struct {
		Durable    bool      `yaml:"durable"`
		AutoDelete bool      `yaml:"autodelete"`
		Nowait     bool      `yaml:"nowait"`
		Exclusive  bool      `yaml:"exclusive"`
		Bindings   []Binding `yaml:"bindings"`
	}

	Settings struct {
		Exchanges map[string]Exchange  `yaml:"exchanges"`
		Queues    map[string]QueueSpec `yaml:"queues"`
	}
)

type Connection struct {
	c *amqp.Connection
}

func Connect(url string) (*Connection, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Info("消息队列连接失败:%v\n", err.Error())
		return nil, err
	}
	return &Connection{conn}, nil
}

func DecodeYaml(r io.Reader) (Settings, error) {
	s := Settings{}
	dec := yaml.NewDecoder(r)
	err := dec.Decode(&s)
	if err != nil {
		return s, err
	}
	return s, nil
}

// CreateScheme creates all exchanges, queues and bindinges between them as specified in yaml string
func (c *Connection) CreateScheme(s Settings) error {
	ch, err := c.c.Channel()
	if err != nil {
		return err
	}

	// Create exchanges according to settings
	for name, e := range s.Exchanges {
		err = ch.ExchangeDeclarePassive(name, e.Type, e.Durable, e.AutoDelete, e.Internal, e.Nowait, nil)
		if err == nil {
			continue
		}
		ch, err = c.c.Channel()
		if err != nil {
			return err
		}

		err = ch.ExchangeDeclare(name, e.Type, e.Durable, e.AutoDelete, e.Internal, e.Nowait, nil)
		if err != nil {
			return err
		}
	}

	// Create queues according to settings
	for name, q := range s.Queues {
		_, err := ch.QueueDeclarePassive(name, q.Durable, q.AutoDelete, q.Exclusive, q.Nowait, nil)
		if err == nil {
			continue
		}

		ch, err = c.c.Channel()
		if err != nil {
			return err
		}

		_, err = ch.QueueDeclare(name, q.Durable, q.AutoDelete, q.Exclusive, q.Nowait, nil)
		if err != nil {
			return err
		}
	}

	// Create bindings only now that everything is setup.
	// (If the bindings were created in one run together with exchanges and queues,
	// it would be possible to create binding to not yet existent queue.
	// This way it's still possible but now is an error on the user side)
	for name, e := range s.Exchanges {
		for _, b := range e.Bindings {
			err = ch.ExchangeBind(name, b.Key, b.Exchange, b.Nowait, nil)
			if err != nil {
				return err
			}
		}
	}

	for name, q := range s.Queues {
		for _, b := range q.Bindings {
			err = ch.QueueBind(name, b.Key, b.Exchange, b.Nowait, nil)
			if err != nil {
				return err
			}
		}
	}

	ch.Close()
	return nil
}

func (c *Connection) DeleteScheme(s Settings) error {
	ch, err := c.c.Channel()
	if err != nil {
		return err
	}

	for name := range s.Exchanges {
		err = ch.ExchangeDelete(name, false, false)
		if err != nil {
			return err
		}
	}

	for name := range s.Queues {
		_, err = ch.QueueDelete(name, false, false, false)
		if err != nil {
			return err
		}
	}
	ch.Close()
	return nil
}

// Close closes connection to RabbitMQ
func (c *Connection) Close() error {
	return c.c.Close()
}

// SendMessage publishes plain text message to an exchange with specific routing key
func (c *Connection) SendMessage(ex, key, msg string) error {
	ch, err := c.c.Channel()
	if err != nil {
		return err
	}

	err = ch.Publish(ex, key, false, false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         []byte(msg),
		})
	if err != nil {
		return err
	}
	return ch.Close()
}

// SendBlob publishes byte blob message to an exchange with specific routing key
func (c *Connection) SendBlob(ex, key string, msg []byte) error {
	ch, err := c.c.Channel()
	if err != nil {
		return err
	}

	err = ch.Publish(ex, key, false, false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/octet-stream",
			Body:         msg,
		})
	if err != nil {
		return err
	}
	return ch.Close()
}

// ProcessQueue calls handler function on each message delivered to a queue
func (c *Connection) ProcessQueue(name string, f func([]byte)) error {
	ch, err := c.c.Channel()
	if err != nil {
		return err
	}
	msgs, err := ch.Consume(name, "", true, false, false, false, nil)
	if err != nil {
		return err
	}
	for d := range msgs {
		f(d.Body)
	}
	return nil
}
