# Verify report — 001 transcribe-url

> 2026-09-26 · clipscribe 0.1.0 · Público: mantenedor

## Resultado

**PASS com ressalvas.** 24 critérios PASS e 4 PARTIAL. Nenhum FAIL. Os PARTIAL têm cobertura por teste, mas não foram exercitados contra o serviço real.

Evidência base:
- `go test ./...`: 57 testes, 0 falhas, offline.
- `go test -tags integration ./internal/tools`: `yt-dlp` real, YouTube e X, passou em 277 s.
- `govulncheck`: sem vulnerabilidades. `staticcheck` e `go vet`: limpos.
- Execuções reais: vídeo TED-Ed `qD0_yWgifDM` (caminho legenda) e post do X `2102050467505430555` (caminho Whisper, gravado no vault).

## Critérios

| AC | Resultado | Evidência |
|---|---|---|
| 1.1 | PASS | TED-Ed real gerou `.md`; `TestManualCaptionInSpokenLanguageSkipsWhisper` |
| 1.2 | PASS | frontmatter real do post do X e do TED-Ed; `TestRenderMarkdownYouTube` |
| 1.3 | PASS | `tags: clippings, transcript` nos dois clippings reais |
| 1.4 | PASS | links `youtu.be/...?t=` no TED-Ed; texto puro no X |
| 1.5 | PASS | TED-Ed real: `using manual captions (en)`, `transcriber: captions-manual` |
| 1.5b | PASS | `TestTranslatedCaptionIsIgnored` |
| 1.6 | PASS | post do X real: `transcriber: whisper-large-v3-turbo`, `audio_sha256` presente |
| 1.7 | PASS | spike: 57 min de áudio PT processados em 118 s (`ffmpeg` 4,9 s + `whisper-cli` 113 s), com os mesmos argumentos dos adapters |
| 2.1 | PASS | post público do X transcrito sem cookies (38 min de vídeo, 82 parágrafos) |
| 2.2 | PASS | `author: "[[@poteto]]"` no clipping real |
| 2.3 | PARTIAL | `TestXPostWithoutVideo` e mapeamento de stderr em `TestYtDlpErrorMapping`; não testado com post real sem vídeo |
| 3.1 | PASS | `TestCookieFile`, `TestYtDlpCookiesOnlyWhenGiven` |
| 3.2 | PASS | `TestCookieFile` (0644 recusado com exit 2) |
| 3.3 | PARTIAL | `TestAuthRequiredSuggestsCookiesWithoutRetry`; as mensagens de login do `yt-dlp` vêm de conhecimento prévio, não de captura real |
| 3.4 | PASS | `TestCookieFile`, `TestYtDlpErrorsNeverShowCookiePath`; defeito plantado 3 da auditoria detectado |
| 4.1 | PASS | `TestInvalidInputNeverCallsSubprocess`; `youtube.com.evil.io` real saiu com exit 2 |
| 4.2 | PASS | `TestInvalidInputNeverCallsSubprocess` |
| 5.1 | PASS | `TestSameVideoTwiceIsSkipped` |
| 5.2 | PASS | `TestForceReplacesExistingClipping`, `TestForceNeverRewritesArchive` |
| 5.3 | PARTIAL | `TestCancelCleansUp` simula cancelamento do contexto; Ctrl-C real não foi exercitado. As execuções reais deixaram o diretório temporário vazio |
| 5.4 | PASS | `TestKeepAudio` |
| 6.1 | PASS | `TestRenderSRT`, `TestOtherFormatsGoToOutDirNotClippings` |
| 6.2 | PASS | `TestRenderText` |
| 6.3 | PASS | `TestOtherFormatsGoToOutDirNotClippings` |
| 7.1 | PASS | `clipscribe doctor` real: 4 itens `ok`; `TestDoctorReportsFixes` |
| 7.2 | PASS | `TestMissingDependency`, `TestMissingModel` |
| 7.3 | PARTIAL | `TestDoctorInstallModel` e testes de `internal/model` com servidor TLS local; o modelo real foi baixado no spike com `curl` + `shasum`, não pelo `doctor` |
| 8.1 | PASS | o `find` do F0 lista o clipping do X; o MD5 não está no manifest, então ele entra como candidato |

## Drift da spec

- D5 (assumption drift): ASSUMPTION 4 (F1 aceita clippings de transcrição) continua `deferred`. Confirmar no próximo pipeline-drain real.
- Estrutura: o plano previa 5 pacotes; a implementação tem 6 (`internal/tools`, por ciclo de import). `plan.md` foi atualizado.

## Observações para a v1.1

- Tempo real do post do X: 648 s de ponta a ponta. A maior parte foi o download HLS do X (~270 KB/s). O Whisper levou cerca de 80 s.
- O `title` do X vem truncado pelo `yt-dlp` (`"lauren - here's how ... ori..."`). Usar `description` como título no X deixaria o frontmatter mais útil.
- O Whisper errou nomes próprios no post do X (o handle `poteto` virou outra palavra). Um `--prompt` com o nome do autor pode ajudar.
