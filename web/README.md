# TightShip web app

The single-page app, built with Vite + React + TypeScript. `npm run build` writes into
`../internal/webui/dist`, which the Go binary embeds; `npm run dev` serves it with `/api`
proxied to a locally running `tightship serve` on 127.0.0.1:8090.

Rules the app lives by: navigation and controls are rendered from `GET /api/v1/me` and never
from a role name; every request goes through `src/api.ts` so a 401 is handled in one place; every
screen state is a URL.
