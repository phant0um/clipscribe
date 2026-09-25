# Tasks — 001 transcribe-url

Legenda: `[P]` = paralelizável com as outras `[P]` do mesmo bloco. `→` = depende de. Teste sempre antes da implementação.

## Fase 0 — Spike (valida ASSUMPTIONS 1–3)

- [x] T001 `brew install yt-dlp whisper-cpp` (precisa de OK do usuário).
- [x] T002 → T001. Baixar `ggml-large-v3-turbo.bin` manualmente e conferir o SHA-256 da research.
- [x] T003 → T001. Rodar `yt-dlp --dump-single-json` em 1 vídeo do YouTube com legenda manual, 1 sem legenda e 1 post público do X. Salvar em `testdata/ytdlp/*.json`.
- [x] T004 → T003. Baixar 1 VTT manual e salvar em `testdata/captions/*.vtt`.
- [x] T005 → T002. Rodar `whisper-cli -oj` num trecho de 2 min em PT e em EN. Salvar em `testdata/whisper/*.json`. Medir tempo de 1 h de áudio (ASSUMPTION 2).
- [x] T006 → T003–T005. Registrar resultado das ASSUMPTIONS 1–3 na spec. Se alguma falhar, parar e voltar para Clarify.

Checkpoint: fixtures reais em `testdata/`, ASSUMPTIONS 1–3 validadas.

## Fase 1 — Esqueleto

- [x] T010 → T006. `go mod init github.com/<user>/clipscribe`, `cmd/clipscribe/main.go`, `internal/app/run.go` com `Run` devolvendo 2 para args vazios. Teste: `internal/app/run_test.go`.
- [x] T011 → T010. Makefile ou `justfile` com `test`, `vet`, `lint` (`staticcheck`), `build`.

## Fase 2 — Núcleo puro (US4, US6, parte da US1)

- [x] T020 [P] → T010. Testes da allowlist de URL (válidas, hosts parecidos, http, userinfo, sem ID). `internal/media/url_test.go`.
- [x] T021 → T020. Implementar `media.ParseURL`. `internal/media/url.go`.
- [x] T022 [P] → T010. Testes do parse do JSON do yt-dlp contra `testdata/ytdlp/`. `internal/media/metadata_test.go`.
- [x] T023 → T022. Implementar `media.ParseVideo`. `internal/media/metadata.go`.
- [x] T024 [P] → T010. Testes do parse de VTT e do JSON do whisper. `internal/transcript/parse_test.go`.
- [x] T025 → T024. Implementar parsers. `internal/transcript/parse.go`.
- [x] T026 [P] → T010. Testes de `Paragraphs` com a regra C3 (pausa 1,5 s, teto 45 s, pontuação após 20 s). `internal/transcript/paragraph_test.go`.
- [x] T027 → T026. Implementar `Paragraphs`. `internal/transcript/paragraph.go`.
- [x] T028 [P] → T010. Testes de render md (golden file), srt e txt. Inclui escape de aspas e quebras de linha no frontmatter. `internal/transcript/render_test.go`.
- [x] T029 → T028. Implementar renders. `internal/transcript/render.go`.
- [x] T030 [P] → T010. Testes de nome de arquivo C2 (caracteres proibidos, 80 caracteres, emoji, X, colisão) e de escrita atômica. `internal/vault/file_test.go`.
- [x] T031 → T030. Implementar. `internal/vault/file.go`.
- [x] T032 [P] → T010. Testes de dedup por frontmatter em diretórios temporários. `internal/vault/dedup_test.go`.
- [x] T033 → T032. Implementar. `internal/vault/dedup.go`.

Checkpoint: `go test ./...` verde, sem rede e sem binários externos.

## Fase 3 — Orquestração (US1, US2, US3, US5)

- [ ] T040 → T021–T033. Testes de `app.Run` com fakes: caminho legenda (AC1.5), caminho Whisper (AC1.6), X sem vídeo (AC2.3), URL inválida sem subprocesso (AC4.1), flags inválidas (AC4.2), cookie aberto (AC3.2), falha de auth sugere `--cookies` (AC3.3), cookie nunca em stderr (AC3.4), dedup (AC5.1), `--force` (AC5.2), cancelamento limpa temporários (AC5.3), `--keep-audio` (AC5.4), formatos (AC6.3), dependência ausente (AC7.2). `internal/app/run_test.go`.
- [ ] T041 → T040. Implementar `app.Run`, config e flags. `internal/app/run.go`, `internal/app/config.go`.

Checkpoint: todos os ACs das US1–US6 verdes com fakes.

## Fase 4 — Adapters reais

- [ ] T050 [P] → T041. Testes da montagem de argumentos do yt-dlp (`--ignore-config`, `--` antes da URL, `--cookies` só se passado, nunca `--exec`). `internal/media/ytdlp_test.go`.
- [ ] T051 → T050. Implementar adapter yt-dlp. `internal/media/ytdlp.go`.
- [ ] T052 [P] → T041. Testes de argumentos do ffmpeg e do whisper-cli. `internal/media/ffmpeg_test.go`, `internal/media/whisper_test.go`.
- [ ] T053 → T052. Implementar adapters. `internal/media/ffmpeg.go`, `internal/media/whisper.go`.

## Fase 5 — Doctor e modelo (US7)

- [ ] T060 → T041. Testes do `doctor` com `LookPath` falso (AC7.1) e do download do modelo com `httptest.Server`: hash certo grava, hash errado apaga e sai com 1 (AC7.3). `internal/app/doctor_test.go`, `internal/model/install_test.go`.
- [ ] T061 → T060. Implementar. `internal/app/doctor.go`, `internal/model/install.go`.

## Fase 6 — Integração e verificação

- [ ] T070 → T051–T061. Teste com build tag `integration`: vídeo real do YouTube e post real do X (AC1.7, AC2.1). `internal/app/integration_test.go`.
- [ ] T071 → T070. Rodar `clipscribe` num vídeo real para o vault e confirmar que o F0 do pipeline-drain o detecta (AC8.1).
- [ ] T072 → T070. `fullstack-security-audit` no repositório. Corrigir achados.
- [ ] T073 → T072. README e instalação com `content-synthesis`.
- [ ] T074 → T073. `verify-report.md`: cada AC com PASS, FAIL ou PARTIAL.
- [ ] T075 → T074. `shipping-checklist`.
