package main

import (
	"bytes"
	"errors"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/opensourceways/community-robot-lib/interrupts"
	"github.com/opensourceways/community-robot-lib/kafka"
	"github.com/opensourceways/community-robot-lib/logrusutil"
	"github.com/opensourceways/community-robot-lib/mq"
	liboptions "github.com/opensourceways/community-robot-lib/options"
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

func gatherOptions(fs *flag.FlagSet, args ...string) (options, error) {
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
	config, err := loadConfig(o.service.ConfigFile)
	if err != nil {
		logrus.Fatalf("read config failed, err:%s", err.Error())
	}

	// load hmac
	hmac, err := readHmac(o.hmacSecretFile)
	if err != nil {
		logrus.Fatalf("read hmac failed, err:%s", err.Error())
	}

	// init delivery
	d := delivery{
		hmac: func() string {
			return hmac
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

func readHmac(f string) (string, error) {
	v, err := ioutil.ReadFile(f)
	err1 := os.Remove(f)

	if err2 := MultiErrors(err, err1); err2 != nil {
		return "", err2
	}

	return string(bytes.TrimSpace(v)), nil
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
	v, err := ioutil.ReadFile(file)
	err1 := os.Remove(file)

	if err = MultiErrors(err, err1); err != nil {
		return
	}

	str := string(v)
	if str == "" {
		err = errors.New("missing addresses")

		return
	}

	addresses := parseAddress(str)
	if len(addresses) == 0 {
		err = errors.New("no valid address for kafka")

		return
	}

	if err = kafka.ValidateConnectingAddress(addresses); err != nil {
		return
	}

	cfg.Addresses = addresses

	return
}

func parseAddress(addresses string) []string {
	var reIpPort = regexp.MustCompile(`^((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)\.?\b){4}:[1-9][0-9]*$`)

	v := strings.Split(addresses, ",")
	r := make([]string, 0, len(v))
	for i := range v {
		if reIpPort.MatchString(v[i]) {
			r = append(r, v[i])
		}
	}

	return r
}
