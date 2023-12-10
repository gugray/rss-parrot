package main

import (
	"context"
	"fmt"
	"go.uber.org/fx"
	"io"
	"log"
	"net/http"
	"os"
	"rss_parrot/internal"
)

func main() {
	app := fx.New(
		fx.Provide(
			internal.ProvideConfig,
			newServerConfig,
			internal.NewHTTPServer,
			fx.Annotate(internal.NewMux, fx.ParamTags(`group:"routes"`)),
			//asRoute(internal.NewEchoHandler),
			newWebfingerHandlerConfig,
			asRoute(internal.NewWebfingerHandler),
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
		fx.As(new(internal.Route)),
		fx.ResultTags(`group:"routes"`),
	)
}

func newServerConfig(cfg *internal.Config) internal.ServerConfig {
	return internal.ServerConfig(cfg)
}

func newWebfingerHandlerConfig(cfg *internal.Config) internal.WebfingerHandlerConfig {
	return internal.WebfingerHandlerConfig(cfg)
}

func initLogger(cfg *internal.Config) {
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
