package main

import (
	"errors"

	"github.com/opensourceways/xihe-gitlab-hook-delivery/kafka"
)

const systemHookEventPush = "push"

type configuration struct {
	HMAC    string       `json:"hmac"  required:"true"`
	Kafka   kafka.Config `json:"kafka" required:"true"`
	Default botConfig    `json:"default,omitempty"`
}

func (c *configuration) configFor() *botConfig {
	return &c.Default
}

func (c *configuration) Validate() error {
	if c == nil {
		return nil
	}

	if c.HMAC == "" {
		return errors.New("missing hmac")
	}

	if err := c.Kafka.Validate(); err != nil {
		return err
	}

	return c.Default.validate()
}

func (c *configuration) SetDefault() {
	if c == nil {
		return
	}

	c.Kafka.SetDefault()
}

// botConfig
type botConfig struct {
	Topic string `json:"topic" required:"true"`
}

func (c *botConfig) getTopic(event string) string {
	if event == systemHookEventPush {
		return c.Topic
	}

	return ""
}

func (c *botConfig) validate() error {
	if c.Topic == "" {
		return errors.New("missing topic")
	}

	return nil
}
