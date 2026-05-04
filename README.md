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

## Quick install

Linux and macOS (amd64 / arm64):

```bash
curl -fsSL https://raw.githubusercontent.com/antegral/tok/main/install.sh | sh
```

The Linux binaries are statically linked against musl, so they run on every glibc version (Ubuntu, Debian, Fedora, RHEL/Rocky/Alma, Amazon Linux, Alpine, …) with no library dependencies.

Overrides:

```bash
# pin a specific version
curl -fsSL https://raw.githubusercontent.com/antegral/tok/main/install.sh | VERSION=v1.2.0 sh

# system-wide install (needs sudo)
curl -fsSL https://raw.githubusercontent.com/antegral/tok/main/install.sh | INSTALL_DIR=/usr/local/bin sudo sh
```

The script downloads the appropriate release archive, verifies its SHA-256, and installs `tok` into `~/.local/bin` (default).

## Build from source

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

**Google Gemini (local tokenization for the entire built-in catalog, including 3.x previews):**
```bash
tok google/gemini-2.5-pro README.md
tok google/gemini-3.1-pro-preview README.md   # alias-mapped to gemma3, no key needed
```

**HuggingFace (local tokenization via `<org>/<repo>` format):**
```bash
tok meta-llama/Llama-3.1-8B-Instruct README.md
```

## Supported providers

| Prefix | Backend | Network | API key |
|--------|---------|---------|---------|
| `openai/` | tiktoken-go local + `o200k_base`/`cl100k_base` prefix fallback (→ Responses API REST as last resort) | Local: No, Fallback: rare (only when neither matches) | Rare (`OPENAI_API_KEY`, only when prefix fallback also misses) |
| `anthropic/` | Anthropic REST API | Yes | Yes (`ANTHROPIC_API_KEY`) |
| `google/` | genai/tokenizer local + gemma3 alias fallback (→ REST API as last resort) | Local: No, Fallback: rare (only when alias map misses) | Rare (`GEMINI_API_KEY` or `GOOGLE_API_KEY`, only when alias also misses) |
| `<org>/<repo>` | daulet/tokenizers (HuggingFace Hub) | Yes (model download) | Optional (`HF_TOKEN`) |

For OpenAI, the prefix fallback maps `gpt-5*`/`gpt-4o*`/`gpt-4.1*`/`o1*`/`o3*`/`o4*` to `o200k_base`, and `gpt-3.5*`/`gpt-4*` to `cl100k_base`. New OpenAI release names (e.g. `gpt-5.5`, `gpt-5.4-pro`) tokenize locally without any key.

For Google, the alias fallback maps the 3.x family (`gemini-3.1-pro-preview`, `gemini-3-flash-preview`, `gemini-3.1-flash-lite-preview`, …) onto the SDK's existing `gemma3` vocab via `gemini-3-pro-preview` (Pro tier) and `gemini-2.5-flash` (Flash tier). `google.golang.org/genai/tokenizer`'s source confirms 2.0/2.5/3-pro-preview all share the gemma3 vocab; the Gemini 3.1 Pro model card explicitly states "for architecture see Gemini 3 Pro".

## Environment variables

Set these environment variables before running tok. tok does NOT auto-load `.env` — use your shell or a tool like direnv.

To load from `.env`, run:

```bash
set -a; source .env; set +a
```

| Variable | Required | Purpose |
|----------|----------|---------|
| `ANTHROPIC_API_KEY` | Always (for `anthropic/*` models) | No public Claude tokenizer exists — every count goes through the remote API. Get a key: https://console.anthropic.com/settings/keys |
| `GEMINI_API_KEY` | Rare | Only needed when a Gemini model name matches **neither** `genai/tokenizer`'s table **nor** the gemma3 alias map. Every model in the built-in catalog is covered locally. Get a key: https://aistudio.google.com/apikey |
| `GOOGLE_API_KEY` | Rare | Alternative to `GEMINI_API_KEY`; takes precedence if both are set |
| `HF_TOKEN` | Optional (required for private models) | Token from https://huggingface.co/settings/tokens |
| `OPENAI_API_KEY` | Rare | Only needed when an OpenAI model name matches **neither** `tiktoken-go`'s table **nor** the prefix rules (`gpt-5*`/`gpt-4o*`/`gpt-4.1*`/`gpt-4*`/`gpt-3.5*`/`o1*`/`o3*`/`o4*`). Every model in the built-in catalog is covered locally. Get a key: https://platform.openai.com/api-keys |

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
- `GEMINI_API_KEY (or GOOGLE_API_KEY) environment variable is required for Gemini remote tokenization` — only seen when the model name is unknown to both `genai/tokenizer` and the gemma3 alias map (rare; every catalog model is covered locally)
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
