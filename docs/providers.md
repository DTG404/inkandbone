# Choose a GM provider

The server selects the first matching row at startup. Supply the variables for the mode you want in the environment of the game process. Restart the process after changing providers. The GM writes the narration. Automation handles tasks such as extracting NPCs or updating a sheet.

| Variables, in precedence order | GM | Automation |
| --- | --- | --- |
| `OLLAMA_GM_MODEL` + `ANTHROPIC_API_KEY` | Local Ollama | Anthropic |
| `DEEPSEEK_API_KEY` + `DEEPSEEK_AUTO_MODEL` | Default DeepSeek model | Named DeepSeek model |
| `DEEPSEEK_API_KEY` | DeepSeek | Same provider and model |
| `OPENROUTER_API_KEY` + `OPENROUTER_AUTO_MODEL` | Default OpenRouter model | Named OpenRouter model |
| `OPENROUTER_API_KEY` | OpenRouter | Same provider and model |
| `ANTHROPIC_API_KEY` | Anthropic | Same provider and model |
| `OLLAMA_GM_MODEL` + `OLLAMA_AI_MODEL` | First local model | Second local model |
| `OLLAMA_MODEL` | Named model | Same provider and model |

**Single local or homelab model.** The default Ollama base URL is `http://localhost:11434`. In single-model mode, set `OLLAMA_HOST` to use another reachable Ollama or compatible chat server. Supply a base URL including the scheme. The client appends `/v1/chat/completions`. Use an endpoint and model ID available to the game server.

```bash
OLLAMA_HOST=http://model-host:11434 OLLAMA_MODEL=your-model-id ./ttrpg
```

**Cloud provider.** Inject the appropriate key from your secret store into the server’s environment. For a one-off DeepSeek session, this Bash example reads it privately and limits the variable to the launch:

```bash
(
  read -rsp 'DeepSeek API key: ' DEEPSEEK_API_KEY && printf '\n'
  export DEEPSEEK_API_KEY
  exec ./ttrpg
)
```

With no matching configuration the dashboard still opens, but GM turns require an available provider. The startup log identifies the selected provider. To verify it can answer, submit an action in a session and watch for the GM’s reply. For the story context and rulebook excerpts included in requests, see [security and data flow](security.md).
