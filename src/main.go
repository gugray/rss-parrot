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
			logic.ProvideConfig,
			server.NewHTTPServer,
			fx.Annotate(server.NewMux, fx.ParamTags(`group:"routes"`)),
			logic.NewWebfinger,
			logic.NewUserDirectory,
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

func initLogger(cfg *logic.Config) {
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
