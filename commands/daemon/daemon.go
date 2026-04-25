package daemon

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kardianos/service"

	"github.com/mindoc-org/mindoc/commands"
	"github.com/mindoc-org/mindoc/conf"
	"github.com/mindoc-org/mindoc/pkg/logger"
	"github.com/mindoc-org/mindoc/routers"
)

type Daemon struct {
	config *service.Config
	errs   chan error
}

func NewDaemon() *Daemon {

	config := &service.Config{
		Name:             "mindocd",
		DisplayName:      "MinDoc service",
		Description:      "A document online management program.",
		WorkingDirectory: conf.WorkingDirectory,
		Arguments:        os.Args[1:],
	}

	return &Daemon{
		config: config,
		errs:   make(chan error, 100),
	}
}

func (d *Daemon) Config() *service.Config {
	return d.config
}
func (d *Daemon) Start(s service.Service) error {

	go d.Run()
	return nil
}

func (d *Daemon) Run() {

	commands.ResolveCommand(d.config.Arguments)

	commands.RegisterFunction()

	commands.RegisterAutoLoadConfig()

	f, err := filepath.Abs(os.Args[0])

	if err != nil {
		f = os.Args[0]
	}

	fmt.Printf("MinDoc version => %s\nbuild time => %s\nexecutable => %s\n%s\n", conf.VERSION, conf.BUILD_TIME, f, conf.GO_VERSION)

	// 使用 Gin 引擎启动
	engine := routers.SetupRouter()

	port := conf.GetDefaultString("httpport", "8181")
	port = strings.Trim(port, "\"'")
	bindAddr := conf.GetDefaultString("httpaddr", "")
	listenAddr := ":" + port
	if bindAddr != "" {
		listenAddr = bindAddr + ":" + port
	}
	fmt.Printf("Listening on %s\n", listenAddr)

	if err := http.ListenAndServe(listenAddr, engine); err != nil {
		logger.Error("Server error: ", err)
	}
}

func (d *Daemon) Stop(s service.Service) error {
	if service.Interactive() {
		os.Exit(0)
	}
	return nil
}

func Install() {
	d := NewDaemon()
	d.config.Arguments = os.Args[3:]

	s, err := service.New(d, d.config)

	if err != nil {
		logger.Error("Create service error => ", err)
		os.Exit(1)
	}
	err = s.Install()
	if err != nil {
		logger.Error("Install service error:", err)
		os.Exit(1)
	} else {
		logger.Info("Service installed!")
	}

	os.Exit(0)
}

func Uninstall() {
	d := NewDaemon()
	s, err := service.New(d, d.config)

	if err != nil {
		logger.Error("Create service error => ", err)
		os.Exit(1)
	}
	err = s.Uninstall()
	if err != nil {
		logger.Error("Install service error:", err)
		os.Exit(1)
	} else {
		logger.Info("Service uninstalled!")
	}
	os.Exit(0)
}

func Restart() {
	d := NewDaemon()
	s, err := service.New(d, d.config)

	if err != nil {
		logger.Error("Create service error => ", err)
		os.Exit(1)
	}
	err = s.Restart()
	if err != nil {
		logger.Error("Install service error:", err)
		os.Exit(1)
	} else {
		logger.Info("Service Restart!")
	}
	os.Exit(0)
}
