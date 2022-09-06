package main

import (
	"errors"
	"flag"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/opensourceways/community-robot-lib/config"
	"github.com/opensourceways/community-robot-lib/interrupts"
	"github.com/opensourceways/community-robot-lib/kafka"
	"github.com/opensourceways/community-robot-lib/logrusutil"
	"github.com/opensourceways/community-robot-lib/mq"
	liboptions "github.com/opensourceways/community-robot-lib/options"
	"github.com/opensourceways/community-robot-lib/secret"
	"github.com/opensourceways/community-robot-lib/utils"
	"github.com/sirupsen/logrus"
)

const component = "robot-gitlab-hook-delivery"

type options struct {
	service           liboptions.ServiceOptions
	kafkamqConfigFile string
	hmacSecretFile    string
}

func (o *options) Validate() error {
	return o.service.Validate()
}

func gatherOptions(fs *flag.FlagSet, args ...string) options {
	var o options

	o.service.AddFlags(fs)

	fs.StringVar(
		&o.hmacSecretFile, "hmac-secret-file", "/etc/webhook/hmac",
		"Path to the file containing the HMAC secret.",
	)

	fs.StringVar(
		&o.kafkamqConfigFile, "kafkamq-config-file", "/etc/kafkamq/config.yaml",
		"Path to the file containing config of kafkamq.",
	)

	fs.Parse(args)
	return o
}

func main() {
	logrusutil.ComponentInit(component)

	o := gatherOptions(flag.NewFlagSet(os.Args[0], flag.ExitOnError), os.Args[1:]...)
	if err := o.Validate(); err != nil {
		logrus.WithError(err).Fatal("Invalid options")
	}

	// init kafka
	kafkaCfg, err := loadKafkaConfig(o.kafkamqConfigFile)
	if err != nil {
		logrus.Fatalf("Error loading kfk config, err:%v", err)
	}

	if err := connetKafka(&kafkaCfg); err != nil {
		logrus.Fatalf("Error connecting kfk mq, err:%v", err)
	}

	defer kafka.Disconnect()

	// load config
	configAgent := config.NewConfigAgent(func() config.Config {
		return new(configuration)
	})
	if err := configAgent.Start(o.service.ConfigFile); err != nil {
		logrus.WithError(err).Fatal("Error starting config agent.")
	}

	defer configAgent.Stop()

	// load hmac
	secretAgent := new(secret.Agent)
	if err := secretAgent.Start([]string{o.hmacSecretFile}); err != nil {
		logrus.WithError(err).Fatal("Error starting secret agent.")
	}

	defer secretAgent.Stop()

	// init delivery
	gethmac := secretAgent.GetTokenGenerator(o.hmacSecretFile)
	d := delivery{
		hmac: func() string {
			return string(gethmac())
		},
		getConfig: func() (*configuration, error) {
			_, cfg := configAgent.GetConfig()
			if c, ok := cfg.(*configuration); ok {
				return c, nil
			}

			return nil, errors.New("can't convert to configuration")
		},
	}

	defer d.wait()

	// run
	run(&d, o.service.Port, o.service.GracePeriod)
}

func run(d *delivery, port int, gracePeriod time.Duration) {
	defer interrupts.WaitForGracefulShutdown()

	// Return 200 on / for health checks.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})

	// For /hook, handle a webhook normally.
	http.Handle("/gitlab-hook", d)

	httpServer := &http.Server{Addr: ":" + strconv.Itoa(port)}

	interrupts.ListenAndServe(httpServer, gracePeriod)
}

func connetKafka(cfg *mq.MQConfig) error {
	tlsConfig, err := cfg.TLSConfig.TLSConfig()
	if err != nil {
		return err
	}

	err = kafka.Init(
		mq.Addresses(cfg.Addresses...),
		mq.SetTLSConfig(tlsConfig),
		mq.Log(logrus.WithField("module", "kfk")),
	)
	if err != nil {
		return err
	}

	return kafka.Connect()
}

func loadKafkaConfig(file string) (cfg mq.MQConfig, err error) {
	if err = utils.LoadFromYaml(file, &cfg); err != nil {
		return
	}

	if len(cfg.Addresses) == 0 {
		err = errors.New("missing addresses")

		return
	}

	err = kafka.ValidateConnectingAddress(cfg.Addresses)

	return
}
