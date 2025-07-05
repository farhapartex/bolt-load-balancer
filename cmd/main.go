package core

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/farhapartex/bolt-load-balancer/internal/config"
	"github.com/farhapartex/bolt-load-balancer/internal/core"
	"github.com/farhapartex/bolt-load-balancer/internal/logger"
)

const (
	VERSION           = "1.0.0"
	DefaultConfigFile = "config.yaml"
)

type Application struct {
	config       *config.Config
	loadBalancer *core.LB
	logger       *logger.Logger
}

func (app *Application) parseFlags() (configFile string, showVersion bool, showHelp bool, err error) {
	flag.StringVar(&configFile, "config", DefaultConfigFile, "Path to configuration file")
	flag.StringVar(&configFile, "c", DefaultConfigFile, "Path to configuration file (short)")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.BoolVar(&showVersion, "v", false, "Show version information (short)")
	flag.BoolVar(&showHelp, "help", false, "Show help information")
	flag.BoolVar(&showHelp, "h", false, "Show help information (short)")

	flag.Parse()

	return configFile, showVersion, showHelp, nil
}

func (app *Application) printHelp() {
	fmt.Printf(`Go Load Balancer v%s
		USAGE:
			loadbalancer [OPTIONS]

		OPTIONS:
			-c, --config <FILE>    Path to configuration file (default: %s)
			-v, --version          Show version information
			-h, --help             Show this help message

		EXAMPLES:
			# Start with default configuration
			loadbalancer
			
			# Start with custom configuration file
			loadbalancer -c /path/to/config.yaml
			
			# Show version
			loadbalancer --version

		CONFIGURATION:
			The load balancer uses YAML configuration files. See the example
			configuration file for all available options.

		HEALTH ENDPOINTS:
			GET /health    - Load balancer health status
			GET /status    - Detailed status information

		For more information, visit: https://github.com/yourusername/go-loadbalancer
		`, VERSION, DefaultConfigFile)
}

func (app *Application) loadConfig(configFile string) error {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if configFile == DefaultConfigFile {
			fmt.Printf("Configuration file '%s' not found, using default configuration\n", configFile)
			app.config = config.DefaultConfig()
			return nil
		}
		return fmt.Errorf("configuration file '%s' does not exist", configFile)
	}

	cfg, err := config.LoadFromFile(configFile)
	if err != nil {
		return err
	}

	app.config, err = config.LoadFromEnv(cfg)
	if err != nil {
		return fmt.Errorf("failed to load config from yaml file: %w", err)
	}

	return nil
}

func (app *Application) createLoadBalancer() error {
	lb, err := core.NewLB(app.config)
	if err != nil {
		return err
	}

	app.loadBalancer = lb
	return nil
}

func (app *Application) setupGracefulShutdown(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		app.logger.Info("Received shutdown signal")
		cancel()
	}()
}

func (app *Application) startLoadBalancer(ctx context.Context) error {
	errChan := make(chan error, 1)
	go func() {
		errChan <- app.loadBalancer.Start()
	}()

	select {
	case <-ctx.Done():
		app.logger.Info("Shutdown initiated")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := app.loadBalancer.Stop(shutdownCtx); err != nil {
			app.logger.Errorf("Error during shutdown: %v", err)
			return err
		}

		app.logger.Info("Load balancer stopped gracefully")
		return nil

	case err := <-errChan:
		if err != nil {
			app.logger.Errorf("Load balancer failed to start: %v", err)
			return err
		}
		return nil
	}
}

func (app *Application) Run() error {
	configFile, showVersion, showHelp, err := app.parseFlags()
	if err != nil {
		return err
	}
	if showVersion {
		fmt.Printf("Go Load Balancer v%s\n", VERSION)
		return nil
	}

	if showHelp {
		app.printHelp()
		return nil
	}

	if err := app.loadConfig(configFile); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	app.logger = logger.NewLogger(app.config.Logging)
	app.logger.Infof("Starting Go Load Balancer v%s", VERSION)

	if err := app.createLoadBalancer(); err != nil {
		return fmt.Errorf("failed to create load balancer: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	app.setupGracefulShutdown(cancel)

	return app.startLoadBalancer(ctx)
}
