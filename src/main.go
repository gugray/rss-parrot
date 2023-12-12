package main

import (
	"context"
	"fmt"
	"go.uber.org/fx"
	"io"
	"log"
	"net/http"
	"os"
	"rss_parrot/config"
	"rss_parrot/dal"
	"rss_parrot/logic"
	"rss_parrot/server"
)

func main() {
	app := fx.New(
		fx.Provide(
			config.ProvideConfig,
			server.NewHTTPServer,
			fx.Annotate(server.NewMux, fx.ParamTags(`group:"handler_group"`)),
			logic.NewWebfinger,
			logic.NewUserDirectory,
			logic.NewActivitySender,
			logic.NewHttpSigChecker,
			logic.NewUserRetriever,
			logic.NewOutbox,
			logic.NewInbox,
			dal.NewRepo,
			asHandlerGroupDef(server.NewApubHandlerGroup),
			asHandlerGroupDef(server.NewCmdHandlerGroup),
		),
		fx.Invoke(
			initLogger,
			registerHooks,
			func(*http.Server) {},
		),
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

func initLogger(cfg *config.Config) {
	logFile, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		msg := fmt.Sprintf("Failed to open log file '%v': %v", cfg.LogFile, err)
		log.Fatal(msg)
	}
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.SetOutput(io.MultiWriter(os.Stdout, logFile))
}

func registerHooks(lc fx.Lifecycle) {
	lc.Append(
		fx.Hook{
			OnStart: func(context.Context) error {
				log.Printf("Application starting up")
				return nil
			},
			OnStop: func(context.Context) error {
				log.Println("Application shutting down")
				return nil
			},
		},
	)
}
