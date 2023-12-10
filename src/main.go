package main

import (
	"context"
	"fmt"
	"go.uber.org/fx"
	"io"
	"log"
	"net/http"
	"os"
	"rss_parrot/logic"
	"rss_parrot/server"
)

func main() {
	app := fx.New(
		fx.Provide(
			provideConfig,
			newServerConfig,
			server.NewHTTPServer,
			fx.Annotate(server.NewMux, fx.ParamTags(`group:"routes"`)),
			newWebfingerConfig,
			logic.NewWebfinger,
			newUserDirectoryConfig,
			logic.NewUserDirectory,
			newActivitySenderConfig,
			logic.NewActivitySender,
			asRoute(server.NewWebfingerHandler),
			asRoute(server.NewUsersHandler),
			asRoute(server.NewBeepHandler),
		),
		fx.Invoke(
			initLogger,
			registerHooks,
			func(*http.Server) {},
		),
	)
	app.Run()
}

func asRoute(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(server.Route)),
		fx.ResultTags(`group:"routes"`),
	)
}

func newServerConfig(cfg *config) server.ServerConfig {
	return server.ServerConfig(cfg)
}

func newWebfingerConfig(cfg *config) logic.WebfingerConfig {
	return logic.WebfingerConfig(cfg)
}

func newUserDirectoryConfig(cfg *config) logic.UserDirectoryConfig {
	return logic.UserDirectoryConfig(cfg)
}

func newActivitySenderConfig(cfg *config) logic.ActivitySenderConfig {
	return logic.ActivitySenderConfig(cfg)
}

func initLogger(cfg *config) {
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
