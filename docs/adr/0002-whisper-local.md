# ADR-0002: Transcrição local com whisper.cpp large-v3-turbo

- Status: aceito
- Data: 2026-09-25

## Contexto

O volume é de poucos vídeos por semana, em PT-BR e EN. A máquina alvo é um Apple M5 Pro. APIs na nuvem (OpenAI, Groq, Deepgram) são mais rápidas, mas cobram por minuto, recebem o áudio e exigem guardar uma API key.

## Decisão

Transcrever localmente com `whisper-cli` (pacote Homebrew `whisper-cpp`, aceleração Metal) e o modelo `large-v3-turbo`. Autodetecção de idioma, com `--lang` para forçar.

## Consequências

- Custo zero por vídeo, áudio não sai da máquina, nenhum segredo para gerenciar.
- O modelo ocupa ~1,6 GB em disco e precisa ser baixado e verificado.
- A velocidade depende do hardware. A meta de 5 min por hora de áudio vale para o M5 Pro.
- Um backend de API pode entrar depois pela mesma interface `Transcriber`, sem mudar o domínio.

## Alternativas rejeitadas

- API na nuvem como padrão: custo recorrente e superfície de segurança maior para ganho de latência irrelevante nesse volume.
