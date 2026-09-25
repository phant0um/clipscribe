# Plan — 001 transcribe-url

Base: [spec.md](spec.md), [data-model.md](data-model.md), [research.md](research.md), [constitution](../../memory/constitution.md), ADR-0001 e ADR-0002.

## Fluxo

```
clipscribe <url> [--lang auto|pt|en] [--format md|srt|txt] [--out dir] [--cookies file] [--force] [--keep-audio]
clipscribe doctor [--install-model]

1. parse flags + config            → exit 2 se inválido
2. valida URL (allowlist, https)   → exit 2
3. valida cookie file (0600)       → exit 2
4. checa dependências              → exit 3
5. Probe (yt-dlp --dump-single-json) → Video
6. dedup (platform + video_id em dedup_dirs) → se achou e sem --force: exit 0
7. MkdirTemp + signal.NotifyContext(SIGINT, SIGTERM)
8. YouTube com legenda manual no idioma? → FetchCaptions → parse VTT → Segments
   senão → FetchAudio → ffmpeg 16 kHz mono → sha256 do WAV → whisper-cli → parse JSON → Segments
9. Paragraphs(segments) com a regra C3
10. render (md | srt | txt) → nome (C2) → escrita atômica (.tmp + rename)
11. stdout: caminho final. Temporários removidos (exceto --keep-audio).
```

## Estrutura

```
cmd/clipscribe/main.go          os.Exit(app.Run(ctx, os.Args[1:], app.DefaultDeps()))
internal/app/                   Run, doctor, flags, config, exit codes, Deps
internal/media/                 URL allowlist, Video, parse do JSON do yt-dlp, erros sentinela
internal/tools/                 adapters yt-dlp / ffmpeg / whisper-cli (montagem de args + exec)
internal/transcript/            Segment, parse VTT, parse JSON do whisper, Paragraphs, render md/srt/txt
internal/vault/                 nome de arquivo, dedup por frontmatter, escrita atômica
internal/model/                 download do modelo com verificação SHA-256
testdata/                       saídas reais capturadas no spike (JSON yt-dlp, VTT, JSON whisper)
```

Seis pacotes internos. Os adapters saíram de `media` para `tools` durante a Fase 4, porque o adapter do Whisper importa `transcript`, que já importa `media` (ciclo). `app` é o único que conhece todos os outros. `transcript` e `vault` não importam `os/exec`.

## Interfaces (em `internal/app`)

```go
type Downloader interface {
    Probe(ctx context.Context, url string, opt FetchOpts) (media.Video, error)
    FetchCaptions(ctx context.Context, url, lang, dir string, opt FetchOpts) (path string, err error)
    FetchAudio(ctx context.Context, url, dir string, opt FetchOpts) (path string, err error)
}
type AudioConverter interface {
    ToWAV16k(ctx context.Context, in, out string) error
}
type Transcriber interface {
    Transcribe(ctx context.Context, wav, lang string) (segs []transcript.Segment, detectedLang string, err error)
}
type Deps struct {
    Downloader  Downloader
    Converter   AudioConverter
    Transcriber Transcriber
    LookPath    func(string) (string, error) // exec.LookPath em produção
    Now         func() time.Time
    Stdout, Stderr io.Writer
    ConfigPath  string
}
```

O runner de subprocesso é um só: `exec.CommandContext(ctx, bin, args...)`, stderr capturado com limite de 64 KiB para mensagem de erro, sem herdar stdin.

## Segurança (checklist do plano)

- Allowlist de host após `url.Parse`, com scheme `https` e sem userinfo. `youtu.be/<id>` e `x.com/<user>/status/<id>` aceitos. Hosts parecidos (`youtube.com.evil.io`) rejeitados por comparação exata.
- `--` antes da URL em toda chamada do `yt-dlp`, para a URL nunca ser lida como flag.
- `--ignore-config` em toda chamada. Nenhum `--exec`, `--netrc` ou `--cookies-from-browser`.
- Cookie file: `os.Stat` e `mode.Perm()&0o077 == 0`, e precisa ser arquivo regular. O caminho nunca é impresso.
- Mensagens de erro do `yt-dlp` passam por filtro que remove o caminho do cookie antes de ir para stderr.
- Download do modelo: HTTPS, tamanho esperado conferido, SHA-256 conferido antes do rename. Hash errado apaga o temporário.
- Escrita só em `out_dir`. O nome sanitizado nunca contém separador de caminho. O caminho final é verificado com `filepath.Rel` contra `out_dir`.
- Revisão final com `fullstack-security-audit` antes de fechar a v1.

## Auditoria do plano

- **Over-engineering:** sem framework de CLI, sem lib YAML, sem lib TOML, sem concorrência. As 3 interfaces existem porque cada uma tem implementação real e fake (seam), não por abstração especulativa. Backend de API na nuvem **não** entra agora.
- **Dependências:** zero dependências externas em Go.
- **Constitution:** exit codes, stdout/stderr, escrita atômica, temporários e testes offline estão cobertos. Teste real fica atrás da build tag `integration`.
- **Risco aberto:** as ASSUMPTIONS 1–3 dependem do comportamento real dos binários. Por isso a primeira task é um spike que captura fixtures reais. Se o spike falhar, a spec volta para Clarify.
