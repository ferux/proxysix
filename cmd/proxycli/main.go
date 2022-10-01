package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/bigstats/proxysix/pkg/client"
	"github.com/urfave/cli/v2"
)

const defaultConfigPath = "config.json"

type Config struct {
	Key string `json:"key"`
}

func main() {
	var config Config

	log := log{
		info: &simpleLogger{
			level: "info",
		},
		error: &simpleLogger{
			level: "error",
		},
	}

	app := cli.App{
		Name:  "proxycli",
		Usage: "CLI for querying proxy6 service",
		Flags: []cli.Flag{
			cli.BashCompletionFlag,
			cli.HelpFlag,
			cli.VersionFlag,
			&cli.BoolFlag{
				Name:     "debug",
				Value:    false,
				Category: "support",
				Usage:    "enables debug information",
			},
			&cli.PathFlag{
				Name:      "config",
				Value:     "config.json",
				Category:  "core",
				TakesFile: true,
			},
		},
		Action: cli.ShowAppHelp,
		Before: func(cCtx *cli.Context) error {
			configPath := cCtx.Path("config")
			if configPath == "" {
				return fmt.Errorf("no config path provided")
			}

			payload, err := os.ReadFile(configPath)
			if err != nil {
				return fmt.Errorf("reading file: %w", err)
			}

			err = json.Unmarshal(payload, &config)
			if err != nil {
				return fmt.Errorf("unmarshalling config: %w", err)
			}

			if cCtx.Bool("debug") {
				log.debug = &simpleLogger{
					level: "debug",
				}
			}

			cCtx.Context = withLogger(cCtx.Context, log)

			return nil
		},
		Commands: []*cli.Command{{
			Name: "list",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "desc",
					Usage: "Filters by descr param",
				},
				&cli.StringFlag{
					Name:  "state",
					Usage: "Filters by state",
					Value: "active",
				},
			},
			Action: func(cCtx *cli.Context) error {
				descr := cCtx.String("desc")
				if descr != "" {
					log.Debug("using description filter: %s", descr)
				}

				ctx := cCtx.Context
				proxyClient, err := client.NewHTTPClient(
					ctx, config.Key,
					client.WithLoggerFunc(getLogger),
				)
				if err != nil {
					return fmt.Errorf("making new http client: %w", err)
				}

				state := client.ProxyStateAll
				if stateStr := cCtx.String("state"); stateStr != "" {
					log.Debug("using state %s", stateStr)
					state, err = client.ParseProxyState(stateStr)
					if err != nil {
						return fmt.Errorf("parsing proxy state: %w", err)
					}
				}

				proxies, err := proxyClient.ListProxies(ctx, client.ListProxyParams{
					Descr: descr,
					State: state,
				})
				if err != nil {
					return fmt.Errorf("getting active proxies: %w", err)
				}

				for _, proxy := range proxies {
					proxyJSON, _ := json.Marshal(proxy)
					log.Info("%s", proxyJSON)
				}

				return nil
			},
		}},
	}

	err := app.Run(os.Args)
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
}

type log struct {
	debug *simpleLogger
	info  *simpleLogger
	error *simpleLogger
}

type simpleLogger struct {
	level string
}

func (l *simpleLogger) Log(format string, args ...any) {
	if l == nil {
		return
	}

	fmt.Fprintf(os.Stdout, "[%s] "+format+"\n", append([]any{l.level}, args...)...)
}

func (l log) Debug(format string, args ...any) {
	l.debug.Log(format, args...)
}

func (l log) Info(format string, args ...any) {
	l.info.Log(format, args...)
}

func (l log) Error(format string, args ...any) {
	l.error.Log(format, args...)
}

type ctxKey struct{}

func withLogger(ctx context.Context, log log) context.Context {
	return context.WithValue(ctx, ctxKey{}, log)
}

func getLogger(ctx context.Context) client.Logger {
	log, _ := ctx.Value(ctxKey{}).(log)
	return wrapLogger(log)
}

type simpleWrapper struct {
	log log
}

func (sw simpleWrapper) join(args ...client.KeyValue) string {
	var s strings.Builder
	for _, kv := range args {
		text := fmt.Sprintf("%s=%v", kv.Key, kv.Value)
		s.WriteString(text + " ")
	}

	return s.String()
}

func (sw simpleWrapper) Debug(msg string, args ...client.KeyValue) {
	sw.log.debug.Log(msg + " " + sw.join(args...))
}
func (sw simpleWrapper) Info(msg string, args ...client.KeyValue) {
	sw.log.info.Log(msg + " " + sw.join(args...))
}

func (sw simpleWrapper) Error(msg string, args ...client.KeyValue) {
	sw.log.error.Log(msg + " " + sw.join(args...))
}

func wrapLogger(log log) client.Logger {
	return simpleWrapper{
		log: log,
	}
}
