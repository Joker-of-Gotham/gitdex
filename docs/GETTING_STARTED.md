# Getting Started From Zero

This guide assumes you are starting on a machine that does not yet have the environment required to run `gitdex`.

## 1. Install the Prerequisites

Use the official docs for installation:

- Git:
  https://git-scm.com/book/en/v2/Getting-Started-Installing-Git
- Go:
  https://go.dev/doc/install
- Ollama:
  https://docs.ollama.com/quickstart
- OpenAI API keys:
  https://platform.openai.com/docs/quickstart
- DeepSeek API keys:
  https://api-docs.deepseek.com/
- GitHub SSH setup if you use SSH remotes:
  https://docs.github.com/en/authentication/connecting-to-github-with-ssh

After installation, open a new terminal and verify:

```bash
git --version
go version
```

If you plan to use Ollama, also verify:

```bash
ollama --version
```

If any required command fails, fix that before moving on.

## 2. Choose an AI Provider

You can run `gitdex` in three supported modes:

- Ollama only
- OpenAI only
- DeepSeek only
- Mixed primary + verifier, for example OpenAI primary with Ollama verifier

### Option A: Ollama

Start Ollama, then pull at least one model:

```bash
ollama pull qwen2.5:3b
```

Optional verifier model:

```bash
ollama pull qwen2.5:7b
```

Confirm the models exist:

```bash
ollama list
```

### Option B: OpenAI

Set your API key:

```bash
export OPENAI_API_KEY=your_key_here
```

Windows PowerShell:

```powershell
$env:OPENAI_API_KEY="your_key_here"
```

### Option C: DeepSeek

Set your API key:

```bash
export DEEPSEEK_API_KEY=your_key_here
```

Windows PowerShell:

```powershell
$env:DEEPSEEK_API_KEY="your_key_here"
```

## 3. Clone the Repository

```bash
git clone <your-repo-url>
cd gitdex
```

If you fork the project later, you can rewrite the module path with the helper scripts in `scripts/`.

## 4. Configure `.gitdexrc`

Minimal OpenAI example:

```yaml
llm:
  provider: "openai"
  endpoint: "https://api.openai.com/v1"
  api_key_env: "OPENAI_API_KEY"
  primary:
    provider: "openai"
    model: "gpt-4.1-mini"
    enabled: true
```

Minimal DeepSeek example:

```yaml
llm:
  provider: "deepseek"
  endpoint: "https://api.deepseek.com"
  api_key_env: "DEEPSEEK_API_KEY"
  primary:
    provider: "deepseek"
    model: "deepseek-chat"
    enabled: true
```

Mixed OpenAI primary + Ollama verifier:

```yaml
llm:
  primary:
    provider: "openai"
    model: "gpt-4.1-mini"
    endpoint: "https://api.openai.com/v1"
    api_key_env: "OPENAI_API_KEY"
    enabled: true
  secondary:
    provider: "ollama"
    model: "qwen2.5:7b"
    endpoint: "http://localhost:11434"
    enabled: true
```

See [configs/example.gitdexrc](../configs/example.gitdexrc) for a fuller example.

## 5. Test Before You Run

macOS / Linux:

```bash
make test
```

Windows:

```powershell
.\build.ps1 -Target test
```

This gives you a fast signal that the local environment is usable.

## 6. Build gitdex

macOS / Linux:

```bash
make build
```

Windows:

```powershell
.\build.ps1 -Target build
```

## 7. Launch gitdex

Run from source:

```bash
go run ./cmd/gitdex
```

Run the built binary on macOS / Linux:

```bash
./bin/gitdex
```

Run the built binary on Windows:

```powershell
.\bin\gitdex.exe
```

## 8. First-Run Flow

1. Start `gitdex` inside a real Git repository.
2. Choose the interface language on the first-run language screen.
3. If you use a local provider, select a primary model and optionally a verifier model.
4. Wait for the first AI analysis.
5. Press `o` or `O` to inspect `Workflow`, `Timeline`, `Context`, `Memory`, `Raw`, `Result`, and `Thinking`.
6. Use `[` and `]` to switch scroll focus between panes.
7. Use mouse wheel or `up/down/pgup/pgdn` to scroll the active pane without forcing a huge terminal.
8. Press `g` to set a goal or `f` to choose a workflow.

## 9. Minimum Verification Checklist

Use this as the shortest serious smoke test:

- The app starts without broken icons or mojibake.
- The first-run language selector appears on a fresh config directory.
- After selecting a language, the UI rerenders in that language.
- With Ollama models available, the local model selection screen appears if the configured model is missing.
- Pressing `y` on a view-only advisory marks it reviewed and does not trap the app in a refresh loop.
- The `Thinking` inspector shows provider reasoning when the backend exposes it.
- Mouse wheel and keyboard scrolling work in the main column and the right-side panes.
- `Timeline` and `Result` update after handling a command or advisory.

## 10. Troubleshooting

If `gitdex` opens but AI is disabled:

- Confirm your provider credentials or local runtime are actually available.
- For Ollama, confirm the service is running and the model exists:

```bash
ollama list
```

- For OpenAI / DeepSeek, confirm the API key environment variable is set in the shell that launches `gitdex`.

If Git is not detected:

- Confirm `git` is on `PATH`.
- Restart your terminal after installing Git.

If the language screen does not appear:

- That is expected on a machine that already has a saved config.
- Press `L` from the main screen to reopen language settings.

If you want to publish the project:

- See [DEPLOYMENT.md](DEPLOYMENT.md) for the release architecture
- See [PUBLISHING_TO_GITHUB.md](PUBLISHING_TO_GITHUB.md) for the exact release checklist
