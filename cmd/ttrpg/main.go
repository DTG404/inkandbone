package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/digitalghost404/inkandbone/internal/ai"
	"github.com/digitalghost404/inkandbone/internal/api"
	"github.com/digitalghost404/inkandbone/internal/db"
	mcpserver "github.com/digitalghost404/inkandbone/internal/mcp"
	ttrpgweb "github.com/digitalghost404/inkandbone/web"
	"golang.org/x/term"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		log.Printf("fatal: %v", err)
		os.Exit(1)
	}
}

func run(args []string, stdin *os.File, stdout io.Writer) error {
	rootCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}
	defaultDBPath := filepath.Join(home, ".ttrpg", "ttrpg.db")
	flags := flag.NewFlagSet("ttrpg", flag.ContinueOnError)
	flags.SetOutput(stdout)
	dbFlag := flags.String("db", defaultDBPath, "path to SQLite database file")
	listenFlag := flags.String("listen", "127.0.0.1:7432", "HTTP listen address")
	tlsCertFlag := flags.String("tls-cert", "", "path to TLS certificate file")
	tlsKeyFlag := flags.String("tls-key", "", "path to TLS private key file")
	var allowedOrigins originListFlag
	flags.Var(&allowedOrigins, "allowed-origin", "allowed browser origin (repeatable or comma-separated)")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return fmt.Errorf("parse flags: %w", err)
	}
	securityConfig := api.ListenSecurityConfig{
		AuthSecret:     os.Getenv("TTRPG_AUTH_SECRET"),
		TLSCertFile:    *tlsCertFlag,
		TLSKeyFile:     *tlsKeyFlag,
		AllowedOrigins: allowedOrigins,
	}
	if err := api.ValidateListenSecurity(*listenFlag, securityConfig); err != nil {
		return fmt.Errorf("listen security: %w", err)
	}
	breakerCooldown, err := configuredAutomationBreakerCooldown(os.Getenv("TTRPG_TEST_AUTOMATION_BREAKER_COOLDOWN"))
	if err != nil {
		return fmt.Errorf("automation breaker cooldown: %w", err)
	}

	dbPath := *dbFlag
	dataDir := filepath.Dir(dbPath)

	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	var aiClient ai.Completer
	switch {
	case os.Getenv("OLLAMA_GM_MODEL") != "" && os.Getenv("ANTHROPIC_API_KEY") != "":
		gmModel := os.Getenv("OLLAMA_GM_MODEL")
		aiClient = ai.NewHybridClient(gmModel, os.Getenv("ANTHROPIC_API_KEY"))
		log.Printf("AI: Hybrid (GM=%s via Ollama, automation=Claude Haiku)", gmModel)
	case os.Getenv("DEEPSEEK_API_KEY") != "" && os.Getenv("DEEPSEEK_AUTO_MODEL") != "":
		autoModel := os.Getenv("DEEPSEEK_AUTO_MODEL")
		aiClient = ai.NewDualDeepSeekClient(os.Getenv("DEEPSEEK_API_KEY"), autoModel)
		log.Printf("AI: DeepSeek dual (GM=%s, automation=%s)", ai.DeepSeekModel, autoModel)
	case os.Getenv("DEEPSEEK_API_KEY") != "":
		aiClient = ai.NewDeepSeekClient(os.Getenv("DEEPSEEK_API_KEY"))
		log.Printf("AI: DeepSeek (%s)", ai.DeepSeekModel)
	case os.Getenv("OPENROUTER_API_KEY") != "" && os.Getenv("OPENROUTER_AUTO_MODEL") != "":
		autoModel := os.Getenv("OPENROUTER_AUTO_MODEL")
		aiClient = ai.NewDualOpenRouterClient(os.Getenv("OPENROUTER_API_KEY"), autoModel)
		log.Printf("AI: OpenRouter dual (GM=%s, automation=%s)", ai.OpenRouterModel, autoModel)
	case os.Getenv("OPENROUTER_API_KEY") != "":
		aiClient = ai.NewOpenRouterClient(os.Getenv("OPENROUTER_API_KEY"))
		log.Printf("AI: OpenRouter (%s)", ai.OpenRouterModel)
	case os.Getenv("ANTHROPIC_API_KEY") != "":
		aiClient = ai.NewClient(os.Getenv("ANTHROPIC_API_KEY"))
		log.Println("AI: Anthropic Claude Haiku")
	case os.Getenv("OLLAMA_GM_MODEL") != "" && os.Getenv("OLLAMA_AI_MODEL") != "":
		gmModel := os.Getenv("OLLAMA_GM_MODEL")
		autoModel := os.Getenv("OLLAMA_AI_MODEL")
		aiClient = ai.NewDualOllamaClient(gmModel, autoModel)
		log.Printf("AI: Ollama dual-model (GM=%s, automation=%s)", gmModel, autoModel)
	case os.Getenv("OLLAMA_MODEL") != "":
		model := os.Getenv("OLLAMA_MODEL")
		if baseURL := configuredOllamaURL(os.Getenv("OLLAMA_HOST")); baseURL != "" {
			aiClient = ai.NewOllamaClientWithURL(model, baseURL)
		} else {
			aiClient = ai.NewOllamaClient(model)
		}
		log.Printf("AI: Ollama single-model (%s)", model)
	default:
		log.Println("AI: disabled (set DEEPSEEK_API_KEY, ANTHROPIC_API_KEY, OPENROUTER_API_KEY, or OLLAMA_MODEL)")
	}

	distFS, err := fs.Sub(ttrpgweb.Static, "dist")
	if err != nil {
		return setupFailure(err, database.Close)
	}

	httpServer := api.NewServerWithOptions(database, dataDir, aiClient, api.ServerOptions{
		Security:                  securityConfig,
		RootContext:               rootCtx,
		AutomationBreakerCooldown: breakerCooldown,
	})
	httpServer.RegisterStatic(http.FS(distFS))

	// When stdin is a pipe (MCP client connected), run the MCP stdio transport
	// in a goroutine and block on HTTP. When stdin is a terminal (interactive /
	// smoke-test mode), skip MCP stdio entirely and just block on HTTP.
	mcpSrv := mcpserver.New(database, httpServer.Bus(), aiClient)
	mcpEnabled := !term.IsTerminal(int(stdin.Fd()))

	protocol := "HTTP"
	if securityConfig.TLSCertFile != "" && securityConfig.TLSKeyFile != "" {
		protocol = "HTTPS"
	}
	log.Printf("%s server listening on %s", protocol, *listenFlag)
	return runServices(rootCtx, stopSignals, httpServer, mcpSrv, mcpEnabled, stdin, stdout,
		database.Close, runtimeConfig{
			address:         *listenFlag,
			certFile:        securityConfig.TLSCertFile,
			keyFile:         securityConfig.TLSKeyFile,
			shutdownTimeout: 15 * time.Second,
		})
}

func configuredOllamaURL(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
}

func configuredAutomationBreakerCooldown(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	cooldown, err := time.ParseDuration(value)
	if err != nil {
		return 0, errors.New("must be a duration between 1ms and 1m")
	}
	if cooldown < time.Millisecond || cooldown > time.Minute {
		return 0, errors.New("must be between 1ms and 1m")
	}
	return cooldown, nil
}

type httpLifecycle interface {
	Start(addr, certFile, keyFile string) error
	Shutdown(context.Context) error
	Close() error
}

type mcpLifecycle interface {
	Start(context.Context, io.ReadCloser, io.Writer) error
}

type runtimeConfig struct {
	address         string
	certFile        string
	keyFile         string
	shutdownTimeout time.Duration
}

func runServices(
	rootCtx context.Context,
	cancelRoot context.CancelFunc,
	web httpLifecycle,
	mcpServer mcpLifecycle,
	mcpEnabled bool,
	stdin io.ReadCloser,
	stdout io.Writer,
	closeDatabase func() error,
	config runtimeConfig,
) error {
	if config.shutdownTimeout <= 0 {
		config.shutdownTimeout = 15 * time.Second
	}

	httpDone := make(chan error, 1)
	go func() { httpDone <- web.Start(config.address, config.certFile, config.keyFile) }()
	var mcpDone chan error
	if mcpEnabled {
		mcpDone = make(chan error, 1)
		go func() { mcpDone <- mcpServer.Start(rootCtx, stdin, stdout) }()
	}

	var (
		primaryErr   error
		httpFinished bool
		mcpFinished  = !mcpEnabled
	)
waitForStop:
	for {
		select {
		case <-rootCtx.Done():
			break waitForStop
		case err := <-httpDone:
			httpFinished = true
			if expectedHTTPStop(rootCtx, err) {
				// Expected result of lifecycle cancellation.
			} else if err == nil || errors.Is(err, http.ErrServerClosed) {
				primaryErr = errors.New("HTTP server stopped unexpectedly")
			} else {
				primaryErr = fmt.Errorf("HTTP server: %w", err)
			}
			break waitForStop
		case err := <-mcpDone:
			mcpFinished = true
			mcpDone = nil
			if err != nil {
				primaryErr = fmt.Errorf("MCP server: %w", err)
				break waitForStop
			}
		}
	}

	cancelRoot()
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), config.shutdownTimeout)
	shutdownErr := web.Shutdown(shutdownCtx)
	cancelShutdown()
	var forceCloseErr, forceJoinErr error
	if shutdownErr != nil {
		forceCloseErr = web.Close()
		forceCtx, cancelForce := context.WithTimeout(context.Background(), config.shutdownTimeout)
		forceJoinErr = web.Shutdown(forceCtx)
		cancelForce()
	}

	waitCtx, cancelWait := context.WithTimeout(context.Background(), config.shutdownTimeout)
	defer cancelWait()
	var httpJoinErr, mcpJoinErr error
	if !httpFinished {
		select {
		case err := <-httpDone:
			if err != nil && !expectedHTTPStop(rootCtx, err) {
				httpJoinErr = fmt.Errorf("join HTTP server: %w", err)
			}
		case <-waitCtx.Done():
			httpJoinErr = fmt.Errorf("join HTTP server: %w", waitCtx.Err())
		}
	}
	if !mcpFinished {
		select {
		case err := <-mcpDone:
			if err != nil {
				mcpJoinErr = fmt.Errorf("join MCP server: %w", err)
			}
		case <-waitCtx.Done():
			mcpJoinErr = fmt.Errorf("join MCP server: %w", waitCtx.Err())
		}
	}

	var databaseErr error
	if forceJoinErr == nil && httpJoinErr == nil && mcpJoinErr == nil {
		databaseErr = closeDatabase()
	} else {
		databaseErr = errors.New("database close skipped because lifecycle services did not join")
	}
	return errors.Join(primaryErr, shutdownErr, forceCloseErr, forceJoinErr, httpJoinErr, mcpJoinErr, databaseErr)
}

func expectedHTTPStop(rootCtx context.Context, err error) bool {
	return rootCtx.Err() != nil &&
		(errors.Is(err, http.ErrServerClosed) || errors.Is(err, context.Canceled))
}

func setupFailure(setupErr error, closeDatabase func() error) error {
	return errors.Join(fmt.Errorf("embed sub: %w", setupErr), closeDatabase())
}

type originListFlag []string

func (origins *originListFlag) String() string {
	return strings.Join(*origins, ",")
}

func (origins *originListFlag) Set(value string) error {
	for _, origin := range strings.Split(value, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			*origins = append(*origins, origin)
		}
	}
	return nil
}
