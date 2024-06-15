package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/yaml.v3"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "./configs/config.yaml", "config file path")
	flag.Parse()
	configPaths := []string{configPath, "config.yaml"}
	cfg, err := getConfig(configPaths)
	if err != nil {
		slog.Error("get config error", slog.String("error", err.Error()))
		panic(err)
	}
	period, err := time.ParseDuration(cfg.Period)
	if err != nil {
		slog.Error("parse period error", slog.String("error", err.Error()))
		panic(err)
	}
	ticker := time.NewTicker(period)

	sig := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		for range ticker.C {
			if contains, err := containsIP(cfg.Cidrs); err != nil {
				slog.Error("check ip error", slog.String("error", err.Error()))
			} else {
				if contains {
					slog.Debug("ip matched, do nothing")
				} else {
					slog.Info("ip not matched, run restart commands")
					for _, cmd := range cfg.RestartCommands {
						slog.Debug("run command", slog.String("command", cmd))
						if err := exec.Command("sudo", "bash", "-c", cmd).Run(); err != nil {
							slog.Error("restart network error", slog.String("command", cmd), slog.String("error", err.Error()))
						}
					}
					slog.Info("restart network finished")
				}
			}
		}
	}()

	go func() {
		for {
			s := <-sig
			slog.Debug("refresh stop", slog.String("signal", s.String()))
			ticker.Stop()
			done <- true
		}
	}()
	<-done
	close(done)
}

type Config struct {
	Logger struct {
		*lumberjack.Logger `yaml:",inline"`
		LogLevel           slog.Level `yaml:"log-level"`
	}
	Cidrs           []string `yaml:"cidrs"`
	Period          string   `yaml:"period"`
	RestartCommands []string `yaml:"restart-commands"`
}

func getConfig(pathList []string) (*Config, error) {
	cfg := new(Config)
	for _, path := range pathList {
		if fileExists(path) {
			if data, err := os.ReadFile(path); err != nil {
				return cfg, fmt.Errorf("can not read config file: %w", err)
			} else if err := yaml.Unmarshal(data, cfg); err != nil {
				return cfg, fmt.Errorf("can not parse config file: %w", err)
			} else {
				initLogger(cfg)
				slog.Debug("load config file", slog.String("path", path))
				return cfg, nil
			}
		}
	}
	return cfg, fmt.Errorf("can not find config file")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func initLogger(cfg *Config) {
	var writer io.Writer
	if cfg.Logger.Logger != nil && cfg.Logger.Logger.Filename != "" {
		writer = cfg.Logger.Logger
	} else {
		writer = os.Stdout
	}
	logHandler := slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level: cfg.Logger.LogLevel,
	})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)
}

func containsIP(cidrStrs []string) (bool, error) {
	if len(cidrStrs) == 0 {
		slog.Warn("cidrs is empty")
		return false, nil
	}
	var err error
	cidrs := make([]*net.IPNet, len(cidrStrs))
	for i, cidrStr := range cidrStrs {
		if _, cidrs[i], err = net.ParseCIDR(cidrStr); err != nil {
			return false, err
		}
	}
	inters, err := net.Interfaces()
	if err != nil {
		return false, err
	}
	for _, inter := range inters {
		if inter.Flags&net.FlagUp != 0 {
			addrs, err := inter.Addrs()
			if err != nil {
				slog.Error("get interface address error", slog.String("interface", inter.Name))
				continue
			}
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					for _, cidr := range cidrs {
						if cidr.Contains(ipnet.IP) {
							slog.Debug("ip matched", slog.String("ip", ipnet.IP.String()))
							return true, nil
						}
					}
				}
			}
		}
	}
	return false, nil
}
