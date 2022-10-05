package main

import (
	"errors"

	"k8s.io/apimachinery/pkg/util/sets"
)

const systemHookEventPush = "push"

var (
	systemHookEventTypes = sets.NewString(systemHookEventPush)
)

type configuration struct {
	Default botConfig `json:"default,omitempty"`
}

func (c *configuration) configFor(org, repo string) *botConfig {
	return &c.Default
}

func (c *configuration) Validate() error {
	if c == nil {
		return nil
	}

	return c.Default.validate()
}

func (c *configuration) SetDefault() {
	if c == nil {
		return
	}

	c.Default.setDefault()
}

type botConfig struct {
	SystemHookEvents []string `json:"system_hook"`
	Topic            string   `json:"topic" required:"true"`
	events           sets.String
}

func (c *botConfig) getTopic(event string) string {
	if c.events.Has(event) {
		return c.Topic
	}

	return ""
}

func (c *botConfig) validate() error {
	if c.Topic == "" {
		return errors.New("missing topic")
	}

	if len(c.SystemHookEvents) == 0 {
		return errors.New("missing system_hook")
	}

	if !systemHookEventTypes.HasAll(c.SystemHookEvents...) {
		return errors.New("includes invalid system hook events")
	}

	return nil
}

func (c *botConfig) setDefault() {
	if len(c.SystemHookEvents) > 0 {
		c.events = sets.NewString(c.SystemHookEvents...)
	}
}
