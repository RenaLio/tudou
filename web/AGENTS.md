# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this
repository.

## Project Overview

Tudou LLM Gateway — a Vue 3 admin dashboard frontend for an LLM API gateway. It manages channels
(upstream providers), API tokens, models, request logs, and system configuration.

## Tech Stack

- Vue 3 (Composition API, `<script setup>`) + Vite + TypeScript
- Vue Router with `createWebHistory`
- Pinia + `pinia-plugin-persistedstate` (localStorage persistence)
- `@tanstack/vue-query` for server state caching
- axios for HTTP client
- Tailwind CSS v4 (CSS-first, no `tailwind.config.js`) + SCSS for custom styling
- `@vueuse/motion` (`v-motion` directive) + `motion-v` (Motion component) for animations
- reka-ui for headless UI primitives (Dialog, Tooltip, etc.)
- echarts for data visualization
- dayjs for date manipulation (do not use native `Date` directly)
- dprint for formatting, ESLint + oxlint for linting

## Common Commands

```sh
# Development server (proxies /api to localhost:8080)
bun dev

# Production build (type-check + vite build)
bun run build

# Type-check only
bun run type-check

# Build without type-check
bun run build-only

# Lint (oxlint + eslint)
bun run lint

# Format all files with dprint
bun run format

# Check formatting
bun run format:check
```

## High-Level Architecture

### API Layer (`src/api/`)

All backend communication goes through `src/api/client.ts`, an axios instance with:

- `baseURL: '/api/v1'`
- Request interceptor injecting `Authorization: Bearer <token>` from the auth store
- Response interceptor catching 401 and redirecting to `/login`

Each domain has its own module (`auth.ts`, `channel.ts`, `token.ts`, `model.ts`, `request-log.ts`,
`stats.ts`, `channel-group.ts`) that:

1. Exports request/response interfaces (e.g. `CreateChannelRequest`)
2. Exports API functions returning `response.data.data` (unwrapped from `ApiResponse<T>`)
3. Often exports display helpers like label maps (`CHANNEL_TYPE_LABELS`, `TOKEN_STATUS_LABELS`) and
   formatting utilities

**Pattern to follow**: keep request/response types and API functions in the same file as the module.
Reusable display helpers (format tokens, format cost) live in `stats.ts`.

### Type System (`src/types/index.ts`)

Core entities (`User`, `Channel`, `Token`, `AIModel`, `ChannelGroup`, etc.) are defined here. API
modules extend these with request/response-specific types locally. Common wrappers:

- `ApiResponse<T>` — `{ code, message, data }`
- `ListResponse<T>` — `{ total, items, page, pageSize }`

### State Management (`src/stores/`)

Two Pinia stores, both persisted to `localStorage`:

- `auth.ts` — `token`, `user`, `isAuthenticated`. Calling `logout()` clears state and redirects.
- `theme.ts` — `'dark' | 'light'`. Sets `data-theme` attribute on `<html>` reactively.

### Routing (`src/router/index.ts`)

- `/login` — public
- `/` — `MainLayout` with authenticated children: `dashboard`, `request-logs`, `tokens`, `channels`,
  `models`, `settings`
- Navigation guard: unauthenticated users are redirected to `/login`; authenticated users hitting
  `/login` are redirected to `/dashboard`

### Styling

Three style layers are loaded in `main.ts` in this order:

1. `theme.css` — CSS custom properties (`--color-*`, `--shadow-*`, `--font-*`, `--radius-*`).
   Defines both dark (default) and light (`[data-theme='light']`) themes.
2. `tailwind.css` — `@import 'tailwindcss'` (pure CSS, not processed by Sass)
3. `styles/main.scss` — SCSS partials: `_variables.scss` (Sass variables), `_mixins.scss`
   (glass-card, flex-center, hover-lift, etc.), `_base.scss` (body, scrollbar, selection)

**Convention**: Vue components use scoped `<style>` with CSS variables from `theme.css`. Complex
reusable patterns use SCSS mixins. Do not use self-closing HTML tags (`<div />`) in Vue templates —
always use explicit close tags (`<div></div>`).

### Animation (`src/utils/motion.ts`)

Two animation systems coexist:

1. **@vueuse/motion** — registered globally as `MotionPlugin`. Used via `v-motion="presetName"`
   directive throughout views.
2. **motion-v** — imported as components (`motion.div`, `Motion`, `AnimatePresence`). Presets
   prefixed with `mv` (e.g. `mvFadeUp`, `mvHoverLift`) are for motion-v's prop-based API.

### Date Handling

Always use `src/utils/date.ts` (dayjs-based). Do not use `new Date()` or native `Date` methods in
components. Key functions: `toRFC3339`, `formatDateTime`, `startOfDay`, `endOfDay`, `addDays`,
`now`.

**API payload rule**: when sending date/time values to the backend, always use RFC3339 format via
`toRFC3339()`. Stats endpoints (`/stats/user/usage/daily`, `/stats/user/usage/hourly`) and any date
range filters expect `startTime`/`endTime` in this format (e.g. `2026-04-28T00:00:00+08:00`).

### Formatting

- dprint is configured with trailing commas everywhere, single quotes, 120 line width, 2-space
  indent. Running `bun run format` will reformat the entire codebase.
- TypeScript has `noUncheckedIndexedAccess: true` enabled — array/object lookups may require
  non-null assertions (`!`) when the compiler cannot prove existence.

## Development Server Proxy

`vite.config.ts` proxies `/api` to `http://localhost:8080`. The backend API is expected to run there
during development.
