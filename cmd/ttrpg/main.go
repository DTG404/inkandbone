package main

import (
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/digitalghost404/inkandbone/internal/ai"
	"github.com/digitalghost404/inkandbone/internal/api"
	"github.com/digitalghost404/inkandbone/internal/db"
	mcpserver "github.com/digitalghost404/inkandbone/internal/mcp"
	ttrpgweb "github.com/digitalghost404/inkandbone/web"
	"golang.org/x/term"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("home dir: %v", err)
	}
	dbPath := filepath.Join(home, ".ttrpg", "ttrpg.db")
	dataDir := filepath.Join(home, ".ttrpg")

	database, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

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
		aiClient = ai.NewOllamaClient(model)
		log.Printf("AI: Ollama single-model (%s)", model)
	default:
		log.Println("AI: disabled (set DEEPSEEK_API_KEY, ANTHROPIC_API_KEY, OPENROUTER_API_KEY, or OLLAMA_MODEL)")
	}

	httpServer := api.NewServer(database, dataDir, aiClient)

	distFS, err := fs.Sub(ttrpgweb.Static, "dist")
	if err != nil {
		log.Fatalf("embed sub: %v", err)
	}
	httpServer.RegisterStatic(http.FS(distFS))

	// When stdin is a pipe (MCP client connected), run the MCP stdio transport
	// in a goroutine and block on HTTP. When stdin is a terminal (interactive /
	// smoke-test mode), skip MCP stdio entirely and just block on HTTP.
	mcpSrv := mcpserver.New(database, httpServer.Bus(), aiClient)
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		go func() {
			if err := mcpSrv.Start(); err != nil {
				log.Printf("MCP server stopped: %v", err)
			}
		}()
	}

	log.Println("HTTP server listening on :7432")
	if err := httpServer.ListenAndServe(":7432"); err != nil {
		log.Printf("HTTP server stopped: %v", err)
	}
}
