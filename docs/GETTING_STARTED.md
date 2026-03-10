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
- Ollama on Windows:
  https://docs.ollama.com/windows
- Ollama on Linux:
  https://docs.ollama.com/linux
- GitHub SSH setup if you use SSH remotes:
  https://docs.github.com/en/authentication/connecting-to-github-with-ssh

After installation, open a new terminal and verify:

```bash
git --version
go version
ollama --version
```

If any command fails, fix that before moving on.

## 2. Start Ollama and Pull a Model

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

## 3. Clone the Repository

```bash
git clone <your-repo-url>
cd gitdex
```

If you fork the project later, you can rewrite the module path with the helper scripts in `scripts/`.

## 4. Test Before You Run

macOS / Linux:

```bash
make test
```

Windows:

```powershell
.\build.ps1 -Target test
```

This gives you a fast signal that the local environment is usable.

## 5. Build gitdex

macOS / Linux:

```bash
make build
```

Windows:

```powershell
.\build.ps1 -Target build
```

## 6. Launch gitdex

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

## 7. First-Run Flow

1. Start `gitdex` inside a real Git repository.
2. Choose the interface language on the first-run language screen.
3. Select a primary Ollama model.
4. Optionally select a secondary verifier model.
5. Wait for the first AI analysis.
6. Press `o` or `O` to inspect `Workflow`, `Timeline`, `Context`, `Memory`, `Raw`, `Result`, and `Thinking`.
7. Press `g` to set a goal or `f` to choose a workflow.

## 8. Minimum Verification Checklist

Use this as the shortest serious smoke test:

- The app starts without broken icons or mojibake.
- The first-run language selector appears on a fresh config directory.
- After selecting a language, the UI rerenders in that language.
- The model selection screen appears when Ollama has local models.
- Pressing `y` on a view-only advisory marks it reviewed and does not trap the app in a refresh loop.
- Pressing `L` from the main screen reopens language settings.
- `Timeline` and `Result` update after handling a command or advisory.

## 9. Troubleshooting

If `gitdex` opens but AI is disabled:

- Confirm Ollama is running.
- Confirm at least one local model exists:

```bash
ollama list
```

If Git is not detected:

- Confirm `git` is on `PATH`.
- Restart your terminal after installing Git.

If the language screen does not appear:

- That is expected on a machine that already has a saved config.
- Press `L` from the main screen to reopen language settings.

If you want to publish the project:

- See [DEPLOYMENT.md](DEPLOYMENT.md) for the release architecture
- See [PUBLISHING_TO_GITHUB.md](PUBLISHING_TO_GITHUB.md) for the exact `v1.0.0` checklist
