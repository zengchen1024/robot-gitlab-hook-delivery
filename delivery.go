package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/opensourceways/robot-gitlab-lib/client"
	"github.com/sirupsen/logrus"
	"github.com/xanzy/go-gitlab"

	"github.com/opensourceways/xihe-gitlab-hook-delivery/kafka"
)

var systemHookMap = map[string]string{
	systemHookEventPush: string(gitlab.EventTypePush),
}

type delivery struct {
	wg        sync.WaitGroup
	hmac      func() string
	getConfig func() (*configuration, error)
}

func (d *delivery) wait() {
	d.wg.Wait()
}

func (d *delivery) getTopic(owner, repo, event string) string {
	if cfg, err := d.getConfig(); err == nil {
		if v := cfg.configFor(owner, repo); v != nil {
			return v.getTopic(event)
		}
	}

	return ""
}

// ServeHTTP validates an incoming webhook and puts it into the event channel.
func (d *delivery) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	eventType, eventGUID, _, payload, ok, _ := client.ValidateWebhook(w, r, d.hmac)
	if !ok {
		return
	}

	fmt.Fprint(w, "Event received. Have a nice day.")

	l := logrus.WithFields(
		logrus.Fields{
			"event-type": eventType,
			"event-id":   eventGUID,
		},
	)

	switch eventType {
	case string(gitlab.EventTypeSystemHook):
		if err := d.deliverySystemHook(payload, r.Header, l); err != nil {
			l.Error(err.Error())
		}
	}
}

type eventBody struct {
	ObjectKind string `json:"object_kind"`
	Project    struct {
		Repo  string `json:"name"`
		Owner string `json:"namespace"`
	} `json:"project"`
}

func (d *delivery) deliverySystemHook(payload []byte, h http.Header, l *logrus.Entry) error {
	e := new(eventBody)
	if err := json.Unmarshal(payload, e); err != nil {
		return err
	}

	kind := strings.ToLower(e.ObjectKind)
	topic := d.getTopic(e.Project.Owner, e.Project.Repo, kind)
	if topic == "" {
		return errors.New("no match topic")
	}

	header := map[string]string{
		"content-type":        h.Get("content-type"),
		"X-Gitlab-Event":      systemHookMap[e.ObjectKind],
		"X-Gitlab-Event-UUID": h.Get("X-Gitlab-Event-UUID"),
		"X-Gitlab-Token":      h.Get("X-Gitlab-Token"),
		"User-Agent":          "Robot-Gitlab-Hook-Delivery",
	}

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()

		if err := kafka.Publish(topic, header, payload); err != nil {
			l.Errorf("failed to publish msg, err:%v", err)
		} else {
			l.Infof("publish to topic of %s successfully", topic)
		}
	}()

	return nil
}
