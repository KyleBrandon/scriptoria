package server

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	_ "net/http/pprof"

	"github.com/KyleBrandon/scriptoria/internal/config"
	"github.com/KyleBrandon/scriptoria/pkg/server/services/health"
	"github.com/KyleBrandon/scriptoria/pkg/stores"
	"github.com/KyleBrandon/scriptoria/pkg/utils"
	"github.com/joho/godotenv"
)

const (
	DEFAULT_SERVER_PORT          = "8080"
	DEFAULT_CONFIG_FILE_LOCATION = "./config/config.json"
)

type ServerConfig struct {
	mux                *http.ServeMux
	ServerPort         string
	Logger             *slog.Logger
	LoggerLevel        *slog.LevelVar
	LogFile            *os.File
	LogFileLocation    string
	ConfigFileLocation string
	OriginPatterns     []string

	SourceStore stores.Store
	DestStore   stores.Store
}

// Used by "flag" to read command line argument
var (
	cmdLineFlagLogLevel string
)

func init() {
	flag.StringVar(&cmdLineFlagLogLevel, "log_level", config.DefaultLogLevel.String(), "The log level to start the server at")
}

func InitializeServer() error {
	slog.Debug(">>InitializeServer")
	defer slog.Debug("<<InitializeServer")

	cfg, err := initializeServerConfig()
	if err != nil {
		return err
	}

	defer cfg.LogFile.Close()

	cfg.mux = http.NewServeMux()

	health.NewHandler(cfg.mux, cfg.LoggerLevel, cfg.Logger)
	err = cfg.SourceStore.Initialize(cfg.mux)
	if err != nil {
		slog.Error("Failed to initialize the source storage", "error", err)
		os.Exit(1)
	}

	// start the profiler
	go func() {
		err := http.ListenAndServe("localhost:6060", nil)
		if err != nil {
			slog.Info("Profiling server failed to start", "error", err)
		}
	}()

	cfg.runServer()

	return nil
}

func initializeServerConfig() (ServerConfig, error) {
	slog.Debug(">>initalizeServerConfig")
	defer slog.Debug("<<initalizeServerConfig")

	cfg := ServerConfig{}

	// MUST BE FIRST FOR LOGGER
	cfg.readEnvironmentVariables()

	// configure slog
	cfg.configureLogger()

	// load the configuration file and environment settings
	config, err := config.LoadConfigSettings(cfg.ConfigFileLocation)
	if err != nil {
		slog.Error("Failed to load config file", "error", err)
		os.Exit(1)
	}

	cfg.OriginPatterns = config.OriginPatterns

	cfg.SourceStore, err = stores.BuildStore(config.SourceStore)
	if err != nil {
		return ServerConfig{}, err
	}

	cfg.DestStore, err = stores.BuildStore(config.DestStore)
	if err != nil {
		return ServerConfig{}, err
	}

	return cfg, nil
}

func (sc *ServerConfig) readEnvironmentVariables() {
	slog.Debug(">>loadConfiguration")
	defer slog.Debug("<<loadConfiguration")

	// load the environment
	err := godotenv.Load()
	if err != nil {
		slog.Warn("Could not load .env file", "error", err)
	}

	sc.ServerPort = os.Getenv("PORT")
	if len(sc.ServerPort) == 0 {
		sc.ServerPort = DEFAULT_SERVER_PORT
	}

	sc.LogFileLocation = os.Getenv("LOG_FILE_LOCATION")

	sc.ConfigFileLocation = os.Getenv("CONFIG_FILE_LOCATION")
	if len(sc.ConfigFileLocation) == 0 {
		sc.ConfigFileLocation = DEFAULT_CONFIG_FILE_LOCATION
	}
}

func (sc *ServerConfig) configureLogger() {
	slog.Debug(">>configureLogger")
	defer slog.Debug("<<configureLogger")

	// craete a variable to store the current log level
	currentLevel := new(slog.LevelVar)
	slog.Info("log level", "level", cmdLineFlagLogLevel)

	// parse the log level from any passed in command line flag
	level, err := utils.ParseLogLevel(cmdLineFlagLogLevel)
	if err != nil {
		slog.Error("Failed to parse the log level, setting to DefaultLogLevel", "error", err, "log_level", cmdLineFlagLogLevel)
		level = config.DefaultLogLevel
	}

	// set the log level
	currentLevel.Set(level)

	// by default we will write to stderr
	logFile := os.Stderr
	if len(sc.LogFileLocation) != 0 {
		logFile, err = os.OpenFile(sc.LogFileLocation, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			slog.Warn("Failed to open log file: %v", "error", err)
			os.Exit(1)
		}

	}

	// create new text handler for log file
	fileHandler := slog.NewTextHandler(logFile, &slog.HandlerOptions{Level: currentLevel})

	logger := slog.New(fileHandler)

	slog.SetDefault(logger)

	sc.Logger = logger
	sc.LoggerLevel = currentLevel
	sc.LogFile = logFile
}

// runServer will start listening for connections
func (config *ServerConfig) runServer() {
	slog.Debug(">>runServer")
	defer slog.Debug("<<runServer")

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", config.ServerPort),
		Handler: config.mux,
	}

	slog.Info("Starting server", "port", config.ServerPort)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Server failed", "error", err)
	}
}
