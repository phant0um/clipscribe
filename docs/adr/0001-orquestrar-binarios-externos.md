# ADR-0001: Orquestrar yt-dlp, ffmpeg e whisper-cli como subprocessos

- Status: aceito
- Data: 2026-09-25

## Contexto

Baixar vídeo do YouTube e do X exige acompanhar mudanças frequentes de proteção (signature cipher, PO token, guest token). Bibliotecas nativas em Go (`kkdai/youtube`) e Rust (`rusty_ytdl`) quebram com essas mudanças e ficam semanas sem correção. Transcrever exige inferência de modelo, que em Go só existe via cgo.

## Decisão

clipscribe chama `yt-dlp`, `ffmpeg` e `whisper-cli` como subprocessos, cada um atrás de uma interface Go. Argumentos são passados como slice, nunca por shell. `yt-dlp` roda com `--ignore-config`.

## Consequências

- A extração acompanha o ritmo de correção do `yt-dlp` com um `brew upgrade`.
- O binário Go fica sem cgo e compila em segundos.
- O usuário precisa instalar três dependências. O comando `doctor` compensa isso com diagnóstico e instrução de instalação.
- A saída dos binários vira contrato: mudança de formato no JSON do `yt-dlp` ou do `whisper-cli` quebra os parsers. Fixtures em `testdata/` detectam isso.

## Alternativas rejeitadas

- Extração nativa em Go: manutenção contínua de engenharia reversa, fora do objetivo do projeto.
- Rust com `whisper-rs`: build C++ e curva maior sem ganho de desempenho, porque a carga está nos binários.
