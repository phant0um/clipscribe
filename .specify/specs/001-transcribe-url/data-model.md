# Data model — 001 transcribe-url

## Tipos de domínio

```go
type Platform string // "youtube" | "x"

// Video: metadados normalizados vindos do yt-dlp.
type Video struct {
    Platform    Platform
    ID          string
    URL         string        // URL canônica (webpage_url)
    Title       string
    Author      string        // YouTube: canal. X: handle sem "@".
    Published   time.Time     // zero se desconhecido
    Duration    time.Duration
    Description string
    ManualSubs  []string      // códigos de idioma com legenda manual
}

// Segment: trecho de fala, vindo de legenda ou do Whisper.
type Segment struct {
    Start, End time.Duration
    Text       string
}

type Paragraph struct {
    Start time.Duration
    Text  string
}

type Transcript struct {
    Video       Video
    Lang        string // "pt" | "en" | outro detectado
    Transcriber string // "captions-manual" | "whisper-large-v3-turbo"
    AudioSHA256 string // vazio quando veio de legenda
    Segments    []Segment
}
```

## Config

`~/.config/clipscribe/config.json`. Flags sobrescrevem a config.

```json
{
  "out_dir": "~/Obsidian/meu-vault/00-INBOX/clippings",
  "dedup_dirs": [
    "~/Obsidian/meu-vault/00-INBOX/clippings",
    "~/Obsidian/meu-vault/08-ARCHIVE/approved",
    "~/Obsidian/meu-vault/08-ARCHIVE/disapproved"
  ],
  "model_path": "~/.local/share/clipscribe/models/ggml-large-v3-turbo.bin"
}
```

## Clipping (saída `md`)

```markdown
---
title: "<título>"
source: "<url canônica>"
author:
  - "[[@<handle>]]"          # X
  - "[[<canal>]]"            # YouTube
published: 2026-09-20
created: 2026-09-25
description: "<primeiros ~160 caracteres da descrição ou da fala>"
platform: "youtube"
video_id: "<id>"
duration: "01:02:03"
lang: "pt"
transcriber: "whisper-large-v3-turbo"
audio_sha256: "<hex>"
tags:
  - "clippings"
  - "transcript"
---
[00:00:00](https://youtu.be/<id>?t=0) Texto do primeiro parágrafo…

[00:00:34](https://youtu.be/<id>?t=34) Texto do segundo parágrafo…
```

- `published` fica ausente quando o `yt-dlp` não informa data.
- `audio_sha256` fica ausente quando o texto veio de legenda.
- No X, o timestamp é texto puro `[00:00:34]`.

## Chave de idempotência

`platform` + `video_id`, lidos do frontmatter dos `.md` em `dedup_dirs`.

## Exit codes

| Código | Significado |
|---|---|
| 0 | sucesso, ou clipping já existente (pulado) |
| 1 | erro de execução (rede, vídeo indisponível, autenticação, hash divergente) |
| 2 | uso inválido (flag, URL fora da allowlist, cookie file com permissão aberta) |
| 3 | dependência ausente (`yt-dlp`, `ffmpeg`, `whisper-cli`, modelo) |
