# Routes

DietDaemon serves its API and dashboard from one origin.

## API

All API endpoints live below `/api/v1/`. They are registered before the dashboard handler, so API misses retain API HTTP behavior and never fall back to the React application.

## Dashboard

The embedded dashboard serves these React paths, including their parameterized forms:

- `/`, `/login`, `/register`, `/auth/callback`, `/verify-email`, `/forgot-password`, `/reset-password`, `/magic`
- `/shared/:token`, `/history/:mealID`, `/body/:tab`
- `/chat`, `/log`, `/history`, `/trends`, `/summary`, `/foods`, `/templates`, `/body`, `/goals`
- `/settings`, `/settings/security`, `/settings/link-bot`, `/settings/aliases`, `/settings/aliases/pending`, `/settings/precedence`, `/settings/nudges`, `/settings/backup`, `/settings/ai-key`, `/settings/assistant`, `/settings/deleted-chats`, `/settings/hevy-import`

Server-side matching is deliberately limited to this list. A direct navigation to one of these paths receives the React entry point, including deep links with parameters.

## Fallbacks and failures

An unknown `GET` or `HEAD` request that accepts HTML receives DietDaemon's branded HTML 404 page with HTTP status 404. Missing assets and non-HTML requests receive an ordinary HTTP 404 instead of the React shell. The React app also has a client-side 404 view for paths reached without a server navigation.

Panic recovery is applied at the HTTP-server boundary: API failures render the structured API error envelope, while dashboard and static failures render branded HTML 500 pages.
