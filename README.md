# clipscribe

**English** · [Português](README.pt-BR.md)

> Version 0.2.1 · 2026-09-26 · Audience: people who use or maintain clipscribe

clipscribe transcribes a YouTube or X video and saves the text as a Markdown clipping in the inbox of an Obsidian vault.

The clipping follows the Obsidian Web Clipper schema, so a vault pipeline can ingest it like any other clipping.

## What it does

- Accepts an `https` URL from `youtube.com`, `youtu.be`, `x.com` or `twitter.com`. Any other URL exits with code 2.
- Uses the YouTube manual captions when they are in the spoken language of the video. Automatic and translated captions are ignored.
- Without manual captions, downloads the audio and transcribes it locally with whisper.cpp and the `large-v3-turbo` model.
- Groups speech into paragraphs with timestamps. On YouTube, each timestamp links to that point in the video.
- Skips videos already transcribed, by matching `platform` and `video_id` in the frontmatter of the inbox and archive.
- Writes atomically. The pipeline never sees a half-written clipping.
- Refuses live streams and videos longer than 4 h. On YouTube, the audio download is capped at 4 GB. On X, the audio comes as HLS and yt-dlp does not apply that cap, so duration is the only limit there.

## Requirements

- macOS on Apple Silicon.
- Go 1.26.6 or newer, only to build.
- `yt-dlp`, `ffmpeg` and `whisper-cli`, from Homebrew.
- About 1.6 GB of disk for the model.

## Installation

```bash
brew install yt-dlp ffmpeg whisper-cpp
```

```bash
go install github.com/phant0um/clipscribe/cmd/clipscribe@latest
```

The binary goes to `~/go/bin`. If the shell says `command not found`, add that folder to your PATH:

```bash
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc
```

From a local clone, `make build` builds `./clipscribe`.

```bash
clipscribe doctor --install-model
```

`doctor --install-model` downloads the model from Hugging Face, checks its size and SHA-256, and only then saves it to `~/.local/share/clipscribe/models/`.

## Configuration

Create `~/.config/clipscribe/config.json`:

```json
{
  "out_dir": "~/Obsidian/my-vault/00-INBOX/clippings",
  "dedup_dirs": [
    "~/Obsidian/my-vault/00-INBOX/clippings",
    "~/Obsidian/my-vault/08-ARCHIVE/approved",
    "~/Obsidian/my-vault/08-ARCHIVE/disapproved"
  ]
}
```

| Field | Use | Default |
|---|---|---|
| `out_dir` | Where `md` clippings are saved | none; without it, `md` needs `--out` |
| `dedup_dirs` | Folders searched for a clipping of the same video | `[out_dir]` |
| `model_path` | Path of the whisper model | `~/.local/share/clipscribe/models/ggml-large-v3-turbo.bin` |

Run `clipscribe doctor` to check dependencies, model and config.

## Usage

```bash
clipscribe "https://www.youtube.com/watch?v=qD0_yWgifDM"
```

Quote the URL: zsh treats `?` and `&` as special characters.

Progress goes to stderr. Stdout gets only the path of the file written.

| Flag | Effect |
|---|---|
| `--lang auto\|pt\|en` | **Spoken** language of the video. Default `auto`. clipscribe does not translate. |
| `--format md\|srt\|txt` | Output format. `srt` and `txt` go to the current directory unless `--out` is set. |
| `--out <dir>` | Output directory. |
| `--force` | Transcribes a video again. Replaces the clipping in the inbox; never changes the archive. |
| `--keep-audio` | Keeps the temporary WAV and prints its path. |
| `--cookies <file>` | Dedicated cookie file, for videos that need a login. |

### Videos that need a login

By default, clipscribe only downloads public videos. For a video behind a login, export a cookie file in Netscape format. The file should contain only `x.com` or `youtube.com`, preferably from a secondary account. Then restrict its permissions:

```bash
chmod 600 ~/.config/clipscribe/cookies-x.txt
```

clipscribe refuses the file if its group or other users can read it. The file path never appears in the output. yt-dlp's `--cookies-from-browser` is not used, because it would expose every browser session.

yt-dlp rewrites the cookie file when it finishes, with the session's updated cookies. Check the permissions after the first use.

### Untrusted text

Title, description and transcript come from whoever published the video. clipscribe neutralizes HTML, links, tags and inline code in the clipping, but the text can still contain instructions written to mislead an LLM. Treat clippings as untrusted content in any AI plugin or agent that reads the vault.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success, or video already transcribed |
| 1 | Runtime error: network, video unavailable, login required, model hash mismatch |
| 2 | Invalid usage: flag, URL outside the allowlist, cookie file readable by others |
| 3 | Missing dependency: `yt-dlp`, `ffmpeg`, `whisper-cli` or model |

## Performance

Measured on an Apple M5 Pro with `large-v3-turbo`:

| Step | 57 min of Portuguese audio |
|---|---|
| `ffmpeg` to 16 kHz WAV | 4.9 s |
| `whisper-cli` | 113 s |

Download time depends on the network. In the same test, YouTube took about 3 min to deliver 51 MB.

## Development

```bash
make test
```

```bash
make lint
```

Tests run offline and without the external binaries, using fakes and fixtures in `testdata/`. The project has no Go dependencies outside the standard library.

| Package | Responsibility |
|---|---|
| `internal/app` | Flags, config, orchestration, `doctor` |
| `internal/media` | URL allowlist, yt-dlp metadata |
| `internal/tools` | Adapters for `yt-dlp`, `ffmpeg` and `whisper-cli` |
| `internal/transcript` | VTT and whisper parsers, paragraphs, rendering |
| `internal/vault` | File names, atomic writes, dedup |
| `internal/model` | Verified model download |

## Documents

The spec and ADRs are in Portuguese. The security audit is in English.

- Project rules: [constitution](.specify/memory/constitution.md)
- v1 spec: [spec 001](.specify/specs/001-transcribe-url/spec.md)
- Why subprocesses instead of libraries: [ADR-0001](docs/adr/0001-orquestrar-binarios-externos.md)
- Why local whisper: [ADR-0002](docs/adr/0002-whisper-local.md)
- Security audit: [2026-09-26](docs/security/audit-2026-09-26.md)

## License

MIT. See [LICENSE](LICENSE).
