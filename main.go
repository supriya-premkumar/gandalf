package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strings"
	"syscall"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"

	"github.com/supriya-premkumar/gandalf/api"
	"github.com/supriya-premkumar/gandalf/pkg"
	"github.com/supriya-premkumar/gandalf/types"
)

// GitCommit is the build commit arg
var GitCommit string

var (
	gandalfPort = flag.IntP("port", "p", types.DefaultRESTPort, "Listen port for gandalf")
	serverCrt   = flag.StringP("cert-file", "c", types.DefaultServerCrtPath, "Server Certificate Path")
	serverKey   = flag.StringP("key-file", "k", types.DefaultServerKeyPath, "Server Key Path")
	config      = flag.StringP("config", "f", types.DefaultConfigPath, "gandalf Default Config Path")
)

func main() {
	var baseConfig types.Config
	// Initialize logger
	signalHandler := make(chan os.Signal)
	logger := logrus.New()
	logger.SetFormatter(&nested.Formatter{
		HideKeys:    true,
		FieldsOrder: []string{"component"},
		CustomCallerFormatter: func(f *runtime.Frame) string {
			s := strings.Split(f.Function, ".")
			funcName := s[len(s)-1]
			return fmt.Sprintf(" [%s:%d][%s()]", path.Base(f.File), f.Line, funcName)
		},
	})
	logger.SetReportCaller(true)
	log := logger.WithField("component", types.FixedWidthFormatter("main"))

	// Parse argv
	flag.Usage = func() {
		log.Errorf("Usage: %s [options]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	// verify valid port ranges
	if *gandalfPort < 0 || *gandalfPort > 65535 {
		log.Fatalf("Invalid port specified: %v", *gandalfPort)
	}

	dat, err := ioutil.ReadFile(*config)
	if err != nil {
		log.Fatalf("Failed to load config file %v. Err: %v", *config, err)
	}

	log.Infof("Starting gandalf. \nversion: %s\nconfig: \n%v", GitCommit, string(dat))

	if err := json.Unmarshal(dat, &baseConfig); err != nil {
		log.Fatalf("Failed to unmarshal config file %v. Err: %v", *config, err)
	}

	// Establish a new context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Instantiate admission reviewer and controller interfaces
	adm := pkg.NewAdmissionController(logger, baseConfig)
	ctrl := api.NewRESTServer(ctx, logger, adm, *gandalfPort, *serverCrt, *serverKey)

	if err := ctrl.Start(); err != nil {
		log.Fatalf("gandalf failed to start. Err: %v", err)
	}

	log.Info("gandalf is ready to protect admission requests")

	// Handle TERM gracefully
	signal.Notify(signalHandler, os.Interrupt, syscall.SIGTERM)
	select {
	case sig := <-signalHandler:
		switch sig {
		case syscall.SIGTERM:
			ctrl.Stop()
			os.Exit(0)
		}
	}
}
