---
id: 001
feature: transcribe-url
status: implement
created: 2026-09-25
---

# Spec 001 — Transcrever URL em clipping

## Por quê

Vídeos do YouTube e do X com conteúdo relevante hoje não entram no vault: o Obsidian Web Clipper captura texto, não áudio. O objetivo é transformar poucos vídeos por semana em clippings de texto pesquisáveis, que o pipeline-drain já sabe ingerir.

## O quê

Uma CLI que recebe uma URL e grava a transcrição como clipping em `00-INBOX/clippings/`.

## User stories

### US1 — Transcrever vídeo do YouTube

Como leitor do vault, quero rodar `clipscribe <url-youtube>` e receber um clipping em Markdown com a fala transcrita e timestamps, para buscar e ingerir o conteúdo depois.

Critérios de aceitação:
- AC1.1 Uma URL pública do YouTube gera um arquivo `.md` no diretório de saída configurado.
- AC1.2 O frontmatter segue o schema do Web Clipper (`title`, `source`, `author`, `published`, `created`, `description`, `tags`) mais `platform`, `video_id`, `duration`, `lang`, `transcriber`, `audio_sha256`.
- AC1.3 `tags` contém `clippings` e `transcript`.
- AC1.4 O corpo tem parágrafos, cada um aberto por um timestamp `[hh:mm:ss]` que linka para o ponto do vídeo.
- AC1.5 Se o vídeo tem legenda manual no **idioma falado** (campo `language` do `yt-dlp`, ou `--lang` quando passado), o texto vem da legenda e `transcriber: captions-manual`. Legenda em outro idioma é tradução e é ignorada. Legenda automática é ignorada. Variantes regionais casam pelo prefixo (`pt` casa `pt-BR`, com preferência para `pt-BR`).
- AC1.5b `--lang` indica o idioma falado no vídeo, nunca o idioma de saída. clipscribe não traduz.
- AC1.6 Sem legenda manual, o texto vem do Whisper local e `transcriber: whisper-large-v3-turbo`.
- AC1.7 Um vídeo de 1 h em PT-BR sem legenda é processado (conversão `ffmpeg` + `whisper-cli`) em menos de 5 min num M5 Pro. O tempo de download depende da rede e é medido à parte.

### US2 — Transcrever vídeo do X

Como leitor do vault, quero rodar `clipscribe <url-x>` para um post com vídeo público e receber o mesmo tipo de clipping.

Critérios de aceitação:
- AC2.1 Uma URL pública de post do X com vídeo gera clipping sem cookies.
- AC2.2 `author` usa o formato `"[[@handle]]"`, igual aos clippings do X já existentes.
- AC2.3 Post sem vídeo termina com exit `1` e mensagem clara, sem arquivo gerado.

### US3 — Vídeo que exige login

Como usuário, quero passar um arquivo de cookies dedicado para vídeos que exigem login.

Critérios de aceitação:
- AC3.1 `--cookies <arquivo>` repassa o arquivo ao `yt-dlp`.
- AC3.2 Arquivo com permissão mais aberta que `0600` é recusado com exit `2`.
- AC3.3 Sem `--cookies`, falha de autenticação termina com exit `1` e a mensagem sugere `--cookies`, sem retry.
- AC3.4 Caminho e conteúdo do cookie nunca aparecem em saída, log ou clipping.

### US4 — Entrada inválida

- AC4.1 URL fora da allowlist ou sem `https` termina com exit `2` antes de chamar qualquer subprocesso.
- AC4.2 Flag desconhecida ou valor inválido de `--format`/`--lang` termina com exit `2`.

### US5 — Repetição e interrupção

- AC5.1 Rodar a mesma URL duas vezes não cria segundo arquivo: a segunda execução avisa em stderr, imprime o caminho existente e sai com exit `0`.
- AC5.2 `--force` sobrescreve o clipping existente.
- AC5.3 `Ctrl-C` durante qualquer etapa remove os temporários e não deixa clipping parcial no diretório de saída.
- AC5.4 `--keep-audio` preserva o WAV e imprime seu caminho em stderr.

### US6 — Formatos alternativos

- AC6.1 `--format srt` gera legenda `.srt` válida.
- AC6.2 `--format txt` gera texto corrido sem frontmatter.
- AC6.3 `md` é o padrão. Só `md` vai para o diretório de clippings; `srt` e `txt` vão para o diretório atual, salvo `--out`.

### US7 — Diagnóstico

- AC7.1 `clipscribe doctor` lista `yt-dlp`, `ffmpeg`, `whisper-cli` e o modelo, com versão/status e o comando para corrigir o que falta.
- AC7.2 Qualquer comando que dependa de binário ausente termina com exit `3` e a mesma instrução.
- AC7.3 `clipscribe doctor --install-model` baixa o modelo, confere o SHA-256 fixado e só grava o arquivo final se o hash bater.

### US8 — Integração com o vault

- AC8.1 O clipping gerado é detectado pelo scan F0 do pipeline-drain (`00-INBOX/clippings/*.md`) e segue para F1 sem edição manual.

## Restrições

- macOS em Apple Silicon. Outras plataformas ficam fora da v1.
- Idiomas principais: PT-BR e EN, com autodetecção. `--lang pt|en` força o idioma.
- Fora da v1: diarização, resumo por LLM, tradução, playlist, lote, GUI, backend de API na nuvem.

## Assumptions

- ASSUMPTION 1 (validada 2026-09-25): `yt-dlp` 2026.08.19 leu o post público `x.com/poteto/status/2102050467505430555/video/1` sem cookies (vídeo de 38 min).
- ASSUMPTION 2 (validada 2026-09-25): áudio PT de 57 min 20 s (`MKU9suCNJQo`) levou 4,9 s no `ffmpeg` e 113 s no `whisper-cli` `large-v3-turbo`, com idioma detectado `pt`. Total de ~2 min, abaixo da meta de 5 min. O download do mesmo áudio (51 MB) levou ~3 min por limitação do YouTube.
- ASSUMPTION 3 (validada para YouTube e X em 2026-09-25, com as diferenças do C5): `yt-dlp --dump-single-json` expõe `id`, `title`, `uploader`/`uploader_id`, `upload_date`, `duration`, `description` e a lista de `subtitles` manuais nas duas plataformas.
- ASSUMPTION 4 (deferred, dono: usuário): o F1 triagem aceita clippings de transcrição sem ajuste na skill `triage-classification`. Confirmar no primeiro pipeline-drain real.
- ASSUMPTION 5 (deferred, dono: usuário): o dedup do F0 é por MD5 do conteúdo. Um `--force` que regrava o clipping gera hash novo e o F0 trata como item novo. Aceitável na v1.

## Test seams

- **Seam 1 (principal): `app.Run(ctx, args, deps) int`.** Recebe argumentos e dependências injetadas (Downloader, AudioConverter, Transcriber, Clock, FS) e devolve o exit code. Quase todos os ACs são testados aqui, com fakes, sem rede.
- **Seam 2: parsers puros.** Conversão de JSON do `yt-dlp` em metadados, de VTT/SRT de legenda em segmentos e de saída JSON do `whisper-cli` em segmentos. Testados com fixtures em `testdata/`.
- **Seam 3: adapters de subprocesso.** Testam só a montagem dos argumentos (slice exato), sem executar o binário.
- **Integração real:** atrás da build tag `integration`, cobre AC1.7 e AC2.1.

## Clarifications

- 2026-09-25 (grilling): download via `yt-dlp` em subprocesso; transcrição local via `whisper-cli`; Go; saída em `00-INBOX/clippings`; cookie file opt-in; idempotência por plataforma + ID. Ver `docs/adr/`.
- C1 (2026-09-25): o modelo é baixado por `clipscribe doctor --install-model`, do Hugging Face, com SHA-256 fixado no código. O download vai para arquivo temporário, o hash é conferido e só então o arquivo é movido de forma atômica para `~/.local/share/clipscribe/models/`. Hash divergente apaga o temporário e termina com exit `1`. Novo critério: AC7.3.
- C2 (2026-09-25): nome do arquivo = título sanitizado (remove `/ \ : * ? " < > |` e caracteres de controle, colapsa espaços, corta em 80 caracteres em fronteira de palavra) + `.md`. No X: `@handle — <primeiras palavras do post>`. Colisão de nome com vídeo diferente acrescenta ` (<video_id>)`. A idempotência não usa o nome: procura `platform` + `video_id` no frontmatter dos `.md` em `00-INBOX/clippings/`, `08-ARCHIVE/approved/` e `08-ARCHIVE/disapproved/`, porque o pipeline-drain move clippings processados para `08-ARCHIVE/`. Os diretórios de busca vêm da config (`dedup_dirs`). A busca lê só o bloco de frontmatter de cada arquivo.
- C3 (2026-09-25): novo parágrafo no primeiro destes eventos: pausa > 1,5 s entre segmentos; parágrafo acumulou 45 s; segmento anterior terminou em `.`, `?` ou `!` e o parágrafo já passa de 20 s. Limites são constantes, sem flag na v1. A regra vale para legenda manual e Whisper. Timestamp do YouTube vira link `[hh:mm:ss](https://youtu.be/<id>?t=<s>)`; no X vira texto `[hh:mm:ss]`.
- C4 (2026-09-25, spike T003/T004): o vídeo TED-Ed `qD0_yWgifDM` (áudio em inglês, `language: en`) tem legenda manual `pt-BR` feita por tradutor voluntário. Usar essa legenda como transcrição entregaria tradução, não fala. Regra corrigida em AC1.5 e AC1.5b. Legendas de tradutor trazem linhas de crédito (`Tradutor:`, `Revisor:`), mas essas legendas agora são descartadas.
- C5 (2026-09-25, spike X): diferenças do JSON do X em relação ao YouTube.
  - `id` é o ID da **mídia** (`2101938030122868736`), e `display_id` é o ID do **post** (`2102050467505430555`). `video_id` no clipping e a chave de idempotência usam `display_id` quando existe, senão `id`. Assim a chave bate com o número da URL que o usuário colou.
  - `language` vem `null`. Sem idioma falado conhecido, o atalho de legenda não se aplica no X: a v1 usa o atalho só no YouTube, e o X vai sempre para o Whisper com `-l auto` ou `--lang`.
  - `duration` é float (`2281.984`).
  - `title` vem como `"<nome> - <texto do post truncado>..."`. O nome do arquivo usa a regra C2 do X (`@<uploader_id> — <primeiras palavras de description>`), nunca o `title`.
  - `author` usa `uploader_id` (`poteto`), e o frontmatter fica `"[[@poteto]]"`.
  - A URL do post pode terminar em `/video/<n>`. A allowlist aceita `/<user>/status/<id>` com esse sufixo opcional.
- Seams confirmados sem objeção: `app.Run` como seam principal, parsers puros, adapters só na montagem de argumentos.
