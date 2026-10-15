# SAG Wiki App

A modern wiki application with AI-powered features, built with Vue 3, TypeScript, and Vite, on top of the WeKnora backend.

## Features

- **Wiki System** - Rich text editing with Tiptap, document and folder management
- **Knowledge Q&A** - RAG-based chat with streaming answers and source references
- **Model Configuration** - Configure AI model providers, models, and credentials
- **System Management** - Members & invitations, audit log, platform settings, task queues
- **Account Settings** - Profile and password management

## Tech Stack

- **Frontend**: Vue 3, TypeScript
- **Build**: Vite
- **Routing**: Vue Router
- **State**: Pinia (with persisted state), TanStack Vue Query
- **Styling**: Tailwind CSS
- **UI Components**: shadcn-vue (reka-ui)
- **Editor**: Tiptap 3
- **Validation**: vee-validate + zod
- **Testing**: Vitest

## Getting Started

```bash
# Install dependencies
pnpm install

# Start development server
pnpm dev

# Build for production
pnpm build

# Run tests
pnpm test
```

## Available Scripts

| Command | Description |
|---------|-------------|
| `pnpm dev` | Start development server |
| `pnpm build` | Type-check and build for production |
| `pnpm preview` | Preview production build |
| `pnpm test` | Run tests in watch mode |
| `pnpm test:run` | Run tests once |
| `pnpm lint` | Lint with oxlint |
| `pnpm lint:fix` | Lint and auto-fix |
| `pnpm lint:type` | TypeScript type check |

## Project Structure

```
src/
├── api/                 # API clients (auth, wiki, ai, system)
├── app/                 # App-level providers
├── assets/              # Static assets (logos, icons)
├── components/          # Shared UI components
│   ├── ui/              # Base UI components (shadcn-vue)
│   ├── sag-ui/          # Business-specific components (data-table, tree, pagination, etc.)
│   └── stream-markdown/ # Streaming markdown rendering
├── composables/         # Vue composables
├── hooks/               # Shared hooks
├── layout/              # Root layout components
├── lib/                 # Third-party library configurations
├── pages/               # Page components
│   ├── ai/             # AI pages (knowledge Q&A, model config)
│   ├── common/         # Shared pages (account, app-setting)
│   ├── dashboard/      # Dashboard page
│   ├── errors/         # Error pages (404)
│   ├── login/          # Login page
│   ├── system/         # System pages (members, audit log, settings, queue)
│   └── wiki/           # Wiki pages (knowledge base, editor)
├── router/              # Route definitions and guards
├── stores/              # Pinia stores (auth, app state)
├── test/                # Test setup and utilities
├── types/               # TypeScript type definitions
└── utils/               # Utility functions (request, color, helpers)
```
