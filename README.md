# tok

A CLI tool that counts LLM tokens in text files or stdin. Supports OpenAI, Anthropic Claude, Google Gemini, and HuggingFace models.

## Quick start

```
$ tok openai/gpt-4o README.md
1342

$ tok google/gemini-2.5-pro example.txt
856

$ tok meta-llama/Llama-3.1-8B-Instruct document.md
2104
```

## Installation

Clone the repository and build using the Makefile:

```bash
git clone https://github.com/antegral/tok
cd tok
make build
```

`make build` automatically downloads `lib/libtokenizers.a`, the prebuilt Rust static library required by `daulet/tokenizers`. Supported host platforms: `linux-amd64`, `linux-arm64`, `darwin-amd64` (Intel), `darwin-arm64` (Apple Silicon).

The resulting binary `./tok` is ready to use directly.

### Install on PATH

To use `tok` from anywhere, install a symlink into a directory on your PATH:

```bash
make install                          # default: ~/.local/bin/tok (no sudo)
make install PREFIX=/usr/local        # system-wide (requires sudo)
make install BIN_DIR=/some/dir        # explicit directory
```

`make install` creates a symlink to `./tok` rather than copying — rebuilding (`make build`) is automatically picked up by the installed entry. The Makefile warns if the chosen `BIN_DIR` is not in `PATH`.

### Uninstall

```bash
make uninstall                        # removes ~/.local/bin/tok (or the PREFIX/BIN_DIR you used)
make uninstall PREFIX=/usr/local      # match whatever you installed with
```

Uninstall is idempotent — running it when nothing is installed is safe.

## Usage

Count tokens from a file:

```bash
tok <provider>/<model> <file>
```

Count tokens from stdin:

```bash
tok <provider>/<model> -
```

### Examples by provider

**OpenAI (local tokenization, no API key required):**
```bash
tok openai/gpt-4o README.md
```

**Anthropic Claude (requires API key):**
```bash
tok anthropic/claude-sonnet-4-5 README.md
```

**Google Gemini (local tokenization for most models; requires API key for newer models):**
```bash
tok google/gemini-2.5-pro README.md
```

**HuggingFace (local tokenization via `<org>/<repo>` format):**
```bash
tok meta-llama/Llama-3.1-8B-Instruct README.md
```

## Supported providers

| Prefix | Backend | Network | API key |
|--------|---------|---------|---------|
| `openai/` | tiktoken-go (local BPE) | No | No |
| `anthropic/` | Anthropic REST API | Yes | Yes (ANTHROPIC_API_KEY) |
| `google/` | genai/tokenizer (local) + REST API fallback | Local: No, Fallback: Yes | Only on fallback (GEMINI_API_KEY or GOOGLE_API_KEY) |
| `<org>/<repo>` | daulet/tokenizers (HuggingFace Hub) | Yes (model download) | Optional (HF_TOKEN) |

## Environment variables

Set these environment variables before running tok. tok does NOT auto-load `.env` — use your shell or a tool like direnv.

To load from `.env`, run:

```bash
set -a; source .env; set +a
```

| Variable | Required | Purpose |
|----------|----------|---------|
| `ANTHROPIC_API_KEY` | For `anthropic/*` models | API key from https://console.anthropic.com/settings/keys |
| `GEMINI_API_KEY` | For newer Gemini models (remote tokenization) | API key from https://aistudio.google.com/apikey |
| `GOOGLE_API_KEY` | Alternative to GEMINI_API_KEY | Same source as GEMINI_API_KEY |
| `HF_TOKEN` | Optional (required for private models) | Token from https://huggingface.co/settings/tokens |

**Note:** `OPENAI_API_KEY` is NOT used. OpenAI models use local tiktoken-go tokenization.

## Tab completion

Install shell completion for your shell:

**Bash:**
```bash
tok completion bash | sudo tee /etc/bash_completion.d/tok
```

**Zsh:**
```bash
tok completion zsh > "${fpath[1]}/_tok"
```

**Fish:**
```bash
tok completion fish > ~/.config/fish/completions/tok.fish
```

**PowerShell:**
```bash
tok completion powershell | Out-File -Encoding UTF8 $PROFILE
```

After installation, completion works as follows:

- Empty input or partial provider name: shows `openai/`, `google/`, `anthropic/`
- After `openai/`, `google/`, or `anthropic/`: shows available models from the catalog
- After `<hf-org>/`: queries HuggingFace Hub for models in that org (cached for 24 hours in `~/.cache/tok/hf-orgs/`)

## Error handling

On success, tok outputs a single integer (token count) to stdout and exits with code 0.

On error, tok outputs `error: <message>` to stderr and exits with code 1. Common errors:

- `invalid model spec "foo" (expected <provider>/<model> or <hf-org>/<repo>)` — malformed model specification
- `ANTHROPIC_API_KEY environment variable is required for Claude models` — missing Anthropic API key
- `GEMINI_API_KEY (or GOOGLE_API_KEY) environment variable is required for Gemini remote tokenization` — missing Google API key for unsupported Gemini models
- `HF_TOKEN environment variable is required for private model <org>/<repo>` — missing token for private HuggingFace models
- Standard file I/O errors (file not found, permission denied, etc.)

## Building with CGO

tok uses the `daulet/tokenizers` library, which requires linking against `libtokenizers.a` (a Rust static library). The Makefile automates this.

If you build directly with `go build`, you must set the CGO linker flags:

```bash
CGO_LDFLAGS=-L./lib go build -o tok .
```

The library must be present at `./lib/libtokenizers.a` before building.

## License

See LICENSE file.
