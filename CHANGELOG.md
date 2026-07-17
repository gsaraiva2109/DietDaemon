# Changelog

## [v0.1.0-alpha.3] - 2026-07-15

### Features

- Automatically pull the required Ollama chat and embedding models at startup.

## [v0.1.0-alpha.2] - 2026-07-14

### Features

- Search the imported food catalog and save foods to a personal library.
- Add bulk-import status, MyFitnessPal log import, ingredient-aware suggestions, and shareable links.

### Fixes

- Improve assistant session titles, food matching, and error messages.
- Show individual food-embedding backfill failures for easier recovery.

## [v0.1.0-alpha.1] - 2026-07-13

### Features

- Add an AI meal assistant with chat, food logging, and OpenAI, Anthropic, and Ollama providers.
- Add the embedded dashboard, English and Brazilian Portuguese support, smart reminders, and correction feedback.
- Add bulk food import with unchanged-file skipping.

### Fixes

- Improve authentication routing and chat-session reliability.

### Performance

- Load dashboard routes on demand and self-host fonts for faster page loads.

### Security

- Scope AI chat sessions to their owner and harden authentication handling.
