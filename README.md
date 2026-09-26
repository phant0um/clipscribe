# clipscribe

> Versão 0.2.0 · 2026-09-26 · Público: quem usa ou mantém o clipscribe

clipscribe transcreve um vídeo do YouTube ou do X e grava o texto como clipping Markdown no inbox de um vault Obsidian.

O clipping segue o schema do Obsidian Web Clipper. O pipeline do vault ingere esse arquivo como qualquer outro clipping.

## O que faz

- Aceita uma URL `https` de `youtube.com`, `youtu.be`, `x.com` ou `twitter.com`. Qualquer outra URL sai com exit 2.
- Usa a legenda manual do YouTube quando ela está no idioma falado do vídeo. Legenda automática e legenda traduzida são ignoradas.
- Sem legenda manual, baixa o áudio e transcreve localmente com whisper.cpp e o modelo `large-v3-turbo`.
- Agrupa a fala em parágrafos com timestamp. No YouTube, o timestamp é um link para o ponto do vídeo.
- Pula vídeos já transcritos, procurando `platform` e `video_id` no frontmatter do inbox e do archive.
- Grava de forma atômica. O pipeline nunca vê um clipping pela metade.
- Recusa live e vídeo com mais de 4 h. No YouTube, o download de áudio tem teto de 4 GB. No X, o áudio vem em HLS e o yt-dlp não aplica esse teto; o limite ali é só a duração.

## Requisitos

- macOS em Apple Silicon.
- Go 1.26.6 ou mais novo, só para compilar.
- `yt-dlp`, `ffmpeg` e `whisper-cli`, via Homebrew.
- ~1,6 GB de disco para o modelo.

## Instalação

```bash
brew install yt-dlp ffmpeg whisper-cpp
```

Com o repositório publicado no GitHub:

```bash
go install github.com/phant0um/clipscribe/cmd/clipscribe@latest
```

Antes disso, a partir do clone local:

```bash
make build
```

```bash
clipscribe doctor --install-model
```

`doctor --install-model` baixa o modelo do Hugging Face, confere tamanho e SHA-256 e só então grava em `~/.local/share/clipscribe/models/`.

## Configuração

Crie `~/.config/clipscribe/config.json`:

```json
{
  "out_dir": "~/Obsidian/meu-vault/00-INBOX/clippings",
  "dedup_dirs": [
    "~/Obsidian/meu-vault/00-INBOX/clippings",
    "~/Obsidian/meu-vault/08-ARCHIVE/approved",
    "~/Obsidian/meu-vault/08-ARCHIVE/disapproved"
  ]
}
```

| Campo | Uso | Padrão |
|---|---|---|
| `out_dir` | Onde os clippings `md` são gravados | nenhum; sem ele, `md` exige `--out` |
| `dedup_dirs` | Pastas onde procurar um clipping do mesmo vídeo | `[out_dir]` |
| `model_path` | Caminho do modelo whisper | `~/.local/share/clipscribe/models/ggml-large-v3-turbo.bin` |

Rode `clipscribe doctor` para conferir dependências, modelo e config.

## Uso

```bash
clipscribe "https://www.youtube.com/watch?v=qD0_yWgifDM"
```

O progresso vai para stderr. O stdout recebe só o caminho do arquivo gerado.

| Flag | Efeito |
|---|---|
| `--lang auto\|pt\|en` | Idioma **falado** no vídeo. Padrão `auto`. clipscribe não traduz. |
| `--format md\|srt\|txt` | Formato de saída. `srt` e `txt` vão para o diretório atual, salvo `--out`. |
| `--out <dir>` | Diretório de saída. |
| `--force` | Transcreve de novo um vídeo já existente. Substitui o clipping no inbox; nunca altera o archive. |
| `--keep-audio` | Mantém o WAV temporário e mostra o caminho. |
| `--cookies <arquivo>` | Arquivo de cookies dedicado, para vídeo que exige login. |

### Vídeo que exige login

Por padrão, clipscribe só baixa vídeo público. Para vídeo com login, exporte um arquivo de cookies no formato Netscape. O arquivo deve conter só `x.com` ou `youtube.com`, de preferência de uma conta secundária. Depois, restrinja a permissão:

```bash
chmod 600 ~/.config/clipscribe/cookies-x.txt
```

clipscribe recusa o arquivo se o grupo ou outros usuários puderem lê-lo. O caminho do arquivo nunca aparece na saída. `--cookies-from-browser` do yt-dlp não é usado, porque daria acesso a todas as sessões do navegador.

O yt-dlp regrava o arquivo de cookies ao terminar, com os cookies atualizados da sessão. Confira a permissão depois do primeiro uso.

### Texto não confiável

Título, descrição e transcrição vêm de quem publicou o vídeo. clipscribe neutraliza HTML, links, tags e código inline no clipping, mas o texto pode conter instruções escritas para enganar um LLM. Trate o clipping como conteúdo não confiável em qualquer plugin ou agente de IA que leia o vault.

## Exit codes

| Código | Significado |
|---|---|
| 0 | Sucesso, ou vídeo já transcrito |
| 1 | Erro de execução: rede, vídeo indisponível, login exigido, hash do modelo divergente |
| 2 | Uso inválido: flag, URL fora da allowlist, cookie file com permissão aberta |
| 3 | Dependência ausente: `yt-dlp`, `ffmpeg`, `whisper-cli` ou modelo |

## Desempenho

Medido num Apple M5 Pro com `large-v3-turbo`:

| Etapa | Áudio PT de 57 min |
|---|---|
| `ffmpeg` para WAV 16 kHz | 4,9 s |
| `whisper-cli` | 113 s |

O download depende da rede. No mesmo teste, o YouTube levou ~3 min para entregar 51 MB.

## Desenvolvimento

```bash
make test
```

```bash
make lint
```

Os testes rodam sem rede e sem os binários externos, com fakes e fixtures em `testdata/`. O projeto não tem dependências Go fora da stdlib.

| Pacote | Responsabilidade |
|---|---|
| `internal/app` | Flags, config, orquestração, `doctor` |
| `internal/media` | Allowlist de URL, metadados do yt-dlp |
| `internal/tools` | Adapters de `yt-dlp`, `ffmpeg` e `whisper-cli` |
| `internal/transcript` | Parsers de VTT e whisper, parágrafos, render |
| `internal/vault` | Nome de arquivo, escrita atômica, dedup |
| `internal/model` | Download verificado do modelo |

## Documentos

- Regras do projeto: [constitution](.specify/memory/constitution.md)
- Spec da v1: [spec 001](.specify/specs/001-transcribe-url/spec.md)
- Por que subprocessos em vez de bibliotecas: [ADR-0001](docs/adr/0001-orquestrar-binarios-externos.md)
- Por que whisper local: [ADR-0002](docs/adr/0002-whisper-local.md)
- Auditoria de segurança: [2026-09-26](docs/security/audit-2026-09-26.md)

## Licença

MIT. Veja [LICENSE](LICENSE).
