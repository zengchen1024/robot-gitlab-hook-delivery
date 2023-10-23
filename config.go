package main

import (
	"errors"

	"github.com/opensourceways/xihe-gitlab-hook-delivery/kafka"
)

const systemHookEventPush = "push"

type configuration struct {
	Default botConfig    `json:"default,omitempty"`
	Kafka   kafka.Config `json:"kafka" required:"true"`
	HMAC    string       `json:"hmac"  required:"true"`
}

func (c *configuration) configFor(org, repo string) *botConfig {
	return &c.Default
}

func (c *configuration) Validate() error {
	if c == nil {
		return nil
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

	c.Default.setDefault()

	c.Kafka.SetDefault()
}

type botConfig struct {
	Topic  string `json:"topic" required:"true"`
	events map[string]bool
}

func (c *botConfig) getTopic(event string) string {
	if c.events[event] {
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

func (c *botConfig) setDefault() {
	c.events = map[string]bool{systemHookEventPush: true}
}
