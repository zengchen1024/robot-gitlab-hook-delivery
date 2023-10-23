package main

import (
	"flag"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/opensourceways/server-common-lib/interrupts"
	"github.com/opensourceways/server-common-lib/logrusutil"
	liboptions "github.com/opensourceways/server-common-lib/options"
	"github.com/sirupsen/logrus"

	"github.com/opensourceways/xihe-gitlab-hook-delivery/kafka"
)

const component = "robot-gitlab-hook-delivery"

type options struct {
	service liboptions.ServiceOptions
}

func (o *options) Validate() error {
	return o.service.Validate()
}

func gatherOptions(fs *flag.FlagSet, args ...string) (options, error) {
	var o options

	o.service.AddFlags(fs)

	err := fs.Parse(args)

	return o, err
}

func main() {
	logrusutil.ComponentInit(component)

	o, err := gatherOptions(
		flag.NewFlagSet(os.Args[0], flag.ExitOnError),
		os.Args[1:]...,
	)
	if err != nil {
		logrus.Fatalf("new options failed, err:%s", err.Error())
	}

	if err := o.Validate(); err != nil {
		logrus.WithError(err).Fatal("Invalid options")
	}

	// load config
	config, err := loadConfig(o.service.ConfigFile)
	if err != nil {
		logrus.Fatalf("read config failed, err:%s", err.Error())
	}

	// init kafka
	if err := kafka.Init(&config.Kafka, logrus.NewEntry(logrus.StandardLogger())); err != nil {
		logrus.Error(err)

		return
	}

	defer kafka.Exit()

	// init delivery
	d := delivery{
		hmac: func() string {
			return config.HMAC
		},
		getConfig: func() (*configuration, error) {
			return &config, nil
		},
	}

	defer d.wait()

	// run
	run(&d, o.service.Port, o.service.GracePeriod)
}

func loadConfig(f string) (configuration, error) {
	cfg := configuration{}
	err := LoadFromYaml(f, &cfg)
	err1 := os.Remove(f)

	if err2 := MultiErrors(err, err1); err2 != nil {
		return cfg, err2
	}

	cfg.SetDefault()
	err = cfg.Validate()

	return cfg, err
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
