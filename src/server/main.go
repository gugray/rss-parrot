package main

import (
	"context"
	"fmt"
	"github.com/charmbracelet/log"
	"go.uber.org/fx"
	"io"
	"net/http"
	"os"
	"rss_parrot/dal"
	"rss_parrot/logic"
	"rss_parrot/server"
	"rss_parrot/shared"
	"rss_parrot/texts"
)

type initErrorHandler struct {
}

func (*initErrorHandler) HandleError(err error) {
	fmt.Fprintf(os.Stderr, "Failed to initialize dependency injection\n%v", err)
}

var logger *log.Logger

func main() {

	cfg := shared.LoadConfig()
	provideConfig := func() *shared.Config {
		return cfg
	}

	logger = initLogger(cfg)
	provideLogger := func() shared.ILogger {
		return logger
	}

	app := fx.New(
		fx.NopLogger,
		fx.Provide(
			provideConfig,
			provideLogger,
			server.NewHTTPServer,
			fx.Annotate(server.NewMux, fx.ParamTags(`group:"handler_group"`)),
			logic.NewKeyStore,
			logic.NewMetrics,
			logic.NewFeedFollower,
			logic.NewUserDirectory,
			logic.NewActivitySender,
			logic.NewHttpSigChecker,
			logic.NewUserRetriever,
			logic.NewMessenger,
			logic.NewInbox,
			texts.NewTexts,
			dal.NewRepo,
			asHandlerGroupDef(server.NewApubHandlerGroup),
			asHandlerGroupDef(server.NewApiHandlerGroup),
			asHandlerGroupDef(server.NewWebHandlerGroup),
			asHandlerGroupDef(server.NewMetricsHandlerGroup),
		),
		fx.Invoke(
			registerHooks,
			func(repo dal.IRepo) { repo.InitUpdateDb() },
			func(*http.Server) {},
			test,
		),
		fx.ErrorHook(&initErrorHandler{}),
	)
	app.Run()
}

func asHandlerGroupDef(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(server.IHandlerGroup)),
		fx.ResultTags(`group:"handler_group"`),
	)
}

func initLogger(cfg *shared.Config) *log.Logger {

	logFile, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		msg := fmt.Sprintf("Failed to open log file '%v': %v", cfg.LogFile, err)
		log.Fatal(msg)
	}

	logger := log.New(io.MultiWriter(os.Stdout, logFile))
	logger.SetReportTimestamp(true)
	logger.SetTimeFormat("2006-01-02 15:04:05.000")
	logger.SetOutput(io.MultiWriter(os.Stdout, logFile))
	switch cfg.LogLevel {
	case "Debug":
		logger.SetLevel(log.DebugLevel)
	case "Info":
		logger.SetLevel(log.InfoLevel)
	case "Warn":
		logger.SetLevel(log.WarnLevel)
	case "Error":
		logger.SetLevel(log.ErrorLevel)
	default:
		logger.SetLevel(log.ErrorLevel)
	}
	logger.SetReportCaller(true)

	return logger
}

func registerHooks(lc fx.Lifecycle, metrics logic.IMetrics) {
	lc.Append(
		fx.Hook{
			OnStart: func(context.Context) error {
				logger.Printf("Application starting up")
				metrics.ServiceStarted()
				return nil
			},
			OnStop: func(context.Context) error {
				logger.Printf("Application shutting down")
				return nil
			},
		},
	)
}

func test(ff logic.IFeedFollower, repo dal.IRepo) {
}
