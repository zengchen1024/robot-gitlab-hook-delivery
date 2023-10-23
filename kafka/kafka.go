package kafka

import (
	kfklib "github.com/opensourceways/kafka-lib/agent"
	"github.com/opensourceways/kafka-lib/mq"
)

const deaultVersion = "2.1.0"

type Config struct {
	kfklib.Config
}

func (cfg *Config) SetDefault() {
	if cfg.Version == "" {
		cfg.Version = deaultVersion
	}
}

func Init(cfg *Config, log mq.Logger) error {
	return kfklib.Init(&cfg.Config, log, nil, "", true)
}

var (
	Exit    = kfklib.Exit
	Publish = kfklib.Publish
)
