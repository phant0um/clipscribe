# Research — 001 transcribe-url

Consultado em 2026-09-25.

## Versões

| Item | Versão | Fonte |
|---|---|---|
| Go | 1.26.4 darwin/arm64 | `go version` local |
| yt-dlp | 2026.8.19 | `brew info yt-dlp` |
| whisper.cpp (`whisper-cli`) | 1.9.4 | `brew info whisper-cpp` |
| ffmpeg | 9.0.2 | `brew info ffmpeg` (já instalado) |

## Modelo

Repositório `ggerganov/whisper.cpp` no Hugging Face. SHA-256 = `lfs.oid` da API `https://huggingface.co/api/models/ggerganov/whisper.cpp/tree/main`.

| Arquivo | Bytes | SHA-256 |
|---|---|---|
| `ggml-large-v3-turbo.bin` | 1624555275 | `1fc70f774d38eb169993ac391eea357ef47c88757ef72ee5943879b7e8e2bc69` |
| `ggml-large-v3-turbo-q8_0.bin` | 874188075 | `317eb69c11673c9de1e1f0d459b253999804ec71ac4c23c17ecf5fbe24e259a1` |
| `ggml-large-v3-turbo-q5_0.bin` | 574041195 | `394221709cd5ad1f40c46e6031ca61bce88931e6e088c188294c6d5a55ffa7e2` |

URL de download: `https://huggingface.co/ggerganov/whisper.cpp/resolve/main/<arquivo>`.

Decisão v1: `ggml-large-v3-turbo.bin` (sem quantização), como decidido no grilling. O `q8_0` é a alternativa se o spike mostrar que a meta de 5 min não é atingida.

## Flags usadas

Os itens marcados como "confirmar no spike" dependem das ASSUMPTIONS 1–3 da spec.

### yt-dlp

- Metadados: `--ignore-config --no-playlist --dump-single-json -- <url>`.
- Legenda manual: `--ignore-config --no-playlist --skip-download --write-subs --no-write-auto-subs --sub-langs <código exato> --sub-format vtt -o <dir>/%(id)s.%(ext)s -- <url>`.
- Áudio: `--ignore-config --no-playlist -f bestaudio/best -o <dir>/%(id)s.%(ext)s -- <url>`.
- Cookies (opt-in): `--cookies <arquivo>`.
- Campos do JSON usados: `id`, `title`, `uploader`, `uploader_id`, `channel`, `upload_date` (`YYYYMMDD`), `duration`, `description`, `webpage_url`, `subtitles` (só manuais; `automatic_captions` é ignorado). Confirmar no spike para o X.

Achados do spike (2026-09-25):
- `--sub-langs pt.*` baixaria `pt-BR` e `pt-PT`. clipscribe escolhe o código exato a partir de `subtitles` e passa só ele.
- O JSON completo do `yt-dlp` tem ~780 KB e inclui URLs assinadas do `googlevideo.com` com o IP de quem baixou. Fixtures em `testdata/` guardam só os campos usados. Nunca commitar o JSON bruto.
- O campo `language` existe (`en` no vídeo TED-Ed) e identifica o idioma falado.

### ffmpeg

`-nostdin -hide_banner -loglevel error -i <in> -vn -ac 1 -ar 16000 -c:a pcm_s16le -y <out.wav>`.

### whisper-cli

`-m <modelo> -f <wav> -l <auto|pt|en> -oj -of <dir>/out`, que gera `<dir>/out.json` com `result.language` e `transcription[]` (`offsets.from`/`offsets.to` em ms, `text`). Confirmar formato exato no spike e salvar a saída real em `testdata/`. O padrão do `-l` é `en`, então clipscribe sempre passa `-l` de forma explícita (`auto` quando não há `--lang`).

## Dependências Go

Nenhuma. Tudo stdlib: `flag`, `os/exec`, `context`, `encoding/json`, `text/template`, `crypto/sha256`, `net/http`, `os/signal`.

- Config em JSON (`~/.config/clipscribe/config.json`), porque a stdlib não lê TOML.
- Frontmatter emitido com strings entre aspas duplas escapadas como JSON, o que é YAML válido. Leitura de frontmatter para dedup por varredura de linhas, sem biblioteca YAML.
