package middleware

import (
	"github.com/streadway/amqp"
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
		log.Info("")
	}
}
