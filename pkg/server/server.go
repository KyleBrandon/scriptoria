package server

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	_ "net/http/pprof"

	"github.com/KyleBrandon/scriptoria/internal/config"
	"github.com/KyleBrandon/scriptoria/internal/database"
	"github.com/KyleBrandon/scriptoria/pkg/document/manager"
	"github.com/KyleBrandon/scriptoria/pkg/document/storage"
	"github.com/KyleBrandon/scriptoria/pkg/server/services/health"
	"github.com/KyleBrandon/scriptoria/pkg/utils"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // Import for side effects (PostgreSQL driver)
)

const (
	DEFAULT_SERVER_PORT          = "8080"
	DEFAULT_CONFIG_FILE_LOCATION = "./config/config.json"
)

type ServerConfig struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	mux        *http.ServeMux

	// environment settings
	DatabaseURL        string
	ServerPort         string
	LogFileLocation    string
	ConfigFileLocation string
	Settings           config.Config

	// logging information
	Logger      *slog.Logger
	LoggerLevel *slog.LevelVar
	LogFile     *os.File

	// config file settings
	originPatterns  []string
	sourceStoreType string
	destStoreType   string

	queries         *database.Queries
	DBConnection    *sql.DB
	documentManager *manager.DocumentManager
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

	var err error

	cfg, err := initializeServerConfig()
	if err != nil {
		return err
	}
	defer cfg.LogFile.Close()

	cfg.openDatabase()
	defer cfg.DBConnection.Close()

	cfg.mux = http.NewServeMux()
	cfg.ctx, cfg.cancelFunc = context.WithCancel(context.Background())

	source, err := storage.BuildDocumentStorage(cfg.ctx, cfg.sourceStoreType, cfg.queries, cfg.mux)
	if err != nil {
	}

	destination, err := storage.BuildDocumentStorage(cfg.ctx, cfg.destStoreType, cfg.queries, cfg.mux)
	if err != nil {
		return err
	}

	cfg.documentManager, err = manager.InitializeManager(cfg.ctx, cfg.queries, cfg.mux, source, destination)
	if err != nil {
		slog.Error("Failed to initialize the document manager", "error", err)
		return err
	}
	cfg.documentManager.StartMonitoring()

	// initialize the health endpoint for the server
	health.NewHandler(cfg.mux, cfg.LoggerLevel, cfg.Logger)

	// start the profiler
	go func() {
		slog.Debug("Start profiling server")
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

	cfg.Settings = config

	cfg.originPatterns = config.OriginPatterns
	cfg.sourceStoreType = config.SourceStore
	cfg.destStoreType = config.DestStore

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

	sc.DatabaseURL = os.Getenv("DATABASE_URL")
	if len(sc.DatabaseURL) == 0 {
		slog.Error("no database connection string is configured")
		os.Exit(1)
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

	config.documentManager.CancelAndWait()
}

func (config *ServerConfig) openDatabase() {
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		slog.Error("failed to open database connection", "error", err)
	}

	config.DBConnection = db
	config.queries = database.New(db)
}
