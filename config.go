package main

import (
	"errors"

	"github.com/xanzy/go-gitlab"
	"k8s.io/apimachinery/pkg/util/sets"
)

const systemHookEventPush = "push"

var (
	systemHookEventTypes = sets.NewString(systemHookEventPush)
	webHookEventTypes    = sets.NewString(string(gitlab.EventTypePush))
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
	WebHookEvents    []string `json:"webhook"`
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

	if len(c.SystemHookEvents) > 0 && len(c.WebHookEvents) > 0 {
		return errors.New("don't set system hook and web hook at same time")
	}

	if len(c.SystemHookEvents) > 0 {
		if !systemHookEventTypes.HasAll(c.SystemHookEvents...) {
			return errors.New("includes invalid system hook events")
		}
	}

	if len(c.WebHookEvents) > 0 {
		if !webHookEventTypes.HasAll(c.WebHookEvents...) {
			return errors.New("includes invalid web hook events")
		}
	}

	return nil
}

func (c *botConfig) setDefault() {
	if len(c.SystemHookEvents) > 0 {
		c.events = sets.NewString(c.SystemHookEvents...)

		return
	}

	if len(c.WebHookEvents) > 0 {
		c.events = sets.NewString(c.WebHookEvents...)
	}
}
