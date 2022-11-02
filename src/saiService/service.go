package saiService

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type Service struct {
	Name          string
	Configuration map[string]interface{}
	Handlers      Handler
	Tasks         []func()
	InitTask      func()
	Logger        *zap.Logger
}

var (
	svc  = new(Service)
	eos  = []byte("\n")
	mode string
)

func NewService(name string) *Service {
	svc.Name = name

	return svc
}

func (s *Service) RegisterConfig(path string) {
	yamlData, err := ioutil.ReadFile(path)

	if err != nil {
		log.Printf("yamlErr:  %v", err)
	}

	err = yaml.Unmarshal(yamlData, &s.Configuration)

	if err != nil {
		log.Fatalf("yamlErr: %v", err)
	}
}

func (s *Service) RegisterHandlers(handlers Handler) {
	svc.SetLogger(&mode)
	s.Handlers = handlers
}

func (s *Service) RegisterTasks(tasks []func()) {
	s.Tasks = tasks
}

func (s *Service) RegisterInitTask(initTask func()) {
	s.InitTask = initTask
}

func (s *Service) GetConfig(path string, def interface{}) interface{} {
	steps := strings.Split(path, ".")
	configuration := s.Configuration

	for _, step := range steps {
		val, _ := configuration[step]

		switch val.(type) {
		case map[string]interface{}:
			configuration = val.(map[string]interface{})
			break
		case string:
			return val.(string)
		case int:
			return val.(int)
		case bool:
			return val.(bool)
		default:
			return def
		}
	}

	return def
}

func (s *Service) Start() {
	s.InitTask()

	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:  "start",
				Usage: "Start services",
				Action: func(c *cli.Context) error {
					s.SetLogger(&mode)
					s.StartServices()
					return nil
				},
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "mode",
				Aliases:     []string{"m"},
				Usage:       "set mode (debug or not,if empty)",
				Destination: &mode,
			},
		},
	}

	for method, handler := range s.Handlers {
		command := new(cli.Command)
		command.Name = method
		command.Usage = handler.Description
		command.Action = func(c *cli.Context) error {
			s.SetLogger(&mode)
			err := s.ExecuteCommand(c.Command.Name, c.Args().Slice(), mode) // add args
			if err != nil {
				return fmt.Errorf("error while executing command %s : %w", command.Name, err)
			}
			return nil
		}

		app.Commands = append(app.Commands, command)
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func (s *Service) ExecuteCommand(path string, data []string, mode string) error {
	result, err := s.handleCliCommand(path, data, mode)
	if err != nil {
		return err
	}
	fmt.Println(string(result))
	return nil
}

func (s *Service) StartServices() {

	useHttp := s.GetConfig("common.http.enabled", true).(bool)
	useWS := s.GetConfig("common.ws.enabled", true).(bool)

	if useHttp {
		go s.StartHttp()
	}

	if useWS {
		go s.StartWS()
	}

	s.StartTasks(mode)

	log.Printf("%s has been started!", s.Name)

	s.StartSocket()
}

func (s *Service) StartTasks(mode string) {
	for _, task := range s.Tasks {
		go task()
	}
}

func (s *Service) SetLogger(mode *string) {
	var (
		logger *zap.Logger
		err    error
	)
	if *mode == "debug" {
		logger, err = zap.NewDevelopment(zap.AddStacktrace(zap.DPanicLevel))
		if err != nil {
			log.Fatal("error creating logger : ", err.Error())
		}
		logger.Debug("Logger started", zap.String("mode", "debug"))
	} else {
		logger, err = zap.NewProduction(zap.AddStacktrace(zap.DPanicLevel))
		if err != nil {
			log.Fatal("error creating logger : ", err.Error())
		}
		logger.Info("Logger started", zap.String("mode", "production"))
	}

	s.Logger = logger
}