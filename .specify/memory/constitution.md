# Constitution — clipscribe

Estas regras valem para toda feature. Mudar este arquivo exige issue própria, nunca dentro de uma feature.

## 1. Escopo

- clipscribe transcreve **um** vídeo público do YouTube ou do X por execução e grava um clipping bruto no vault.
- A curadoria (source page, wikilinks, síntese) pertence ao pipeline do vault. clipscribe nunca escreve fora do diretório de saída configurado.

## 2. Código

- Go, stdlib primeiro. Toda dependência externa precisa de justificativa escrita no `plan.md` da feature.
- O trabalho pesado fica em binários externos (`yt-dlp`, `ffmpeg`, `whisper-cli`). clipscribe orquestra; não reimplementa extração, decodificação nem inferência.
- Cada binário externo fica atrás de uma interface pequena. O domínio não importa `os/exec`.
- Erros carregam contexto (`fmt.Errorf("download %s: %w", id, err)`). Sem `panic` fora de `main`.
- `gofmt`, `go vet` e `staticcheck` limpos antes de commit.

## 3. Segurança

- URL de entrada passa por allowlist de host (`youtube.com`, `www.youtube.com`, `m.youtube.com`, `youtu.be`, `x.com`, `twitter.com`, `mobile.twitter.com`) e exige `https`.
- Subprocessos recebem argumentos como slice. Nunca `sh -c`, nunca interpolação em string de shell.
- `yt-dlp` roda sempre com `--ignore-config` e `--no-exec`-equivalente (nenhuma flag `--exec`), e com `--` antes da URL.
- Cookies só via arquivo dedicado passado em `--cookies`, com permissão `0600` ou mais restrita. `--cookies-from-browser` nunca é usado.
- Nenhum segredo, cookie ou caminho de cookie aparece em log, clipping ou mensagem de erro.
- Todo arquivo baixado (modelo) é verificado por SHA-256 antes do uso.

## 4. Testes

- Testes rodam offline e sem os binários externos, usando fakes das interfaces.
- Teste primeiro (red → green → refactor) em cada seam.
- Um teste de integração real existe atrás da build tag `integration` e nunca roda no `go test ./...` padrão.

## 5. Arquivos e estado

- Saída gravada de forma atômica: arquivo temporário no mesmo diretório e depois `os.Rename`.
- Temporários em `os.MkdirTemp`, removidos no fim e em `SIGINT`/`SIGTERM`.
- Idempotência por chave `plataforma + ID do vídeo`: se o clipping já existe, pular, a menos que `--force`.

## 6. UX da CLI

- Exit code `0` sucesso ou pulado, `1` erro de execução, `2` uso inválido (flag, URL fora da allowlist), `3` dependência ausente.
- Progresso em stderr; stdout só recebe o caminho do arquivo gerado.
- Mensagem de erro diz o que fazer a seguir (ex.: `brew install yt-dlp`).

## 7. Performance

- Vídeo de 1 h em PT-BR transcrito em menos de 5 min num Apple M5 Pro com `large-v3-turbo`.
