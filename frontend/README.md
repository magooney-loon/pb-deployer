# pb-deployer Frontend

Modern SvelteKit dashboard for deployment orchestration with real-time WebSocket updates and component-driven architecture.

## Features

- **Real-time Operations**: WebSocket integration for live deployment tracking
- **Component Library**: Comprehensive partial components with consistent design system
- **Type-safe Architecture**: TypeScript throughout with validated interfaces
- **Responsive Design**: Mobile-first approach with Tailwind CSS
- **Performance Optimized**: SvelteKit SSR with adaptive rendering
- **State Management**: Reactive stores with persistent session handling

## Quick Start

```bash
# Install dependencies + development build
go run cmd/scripts/main.go --install

# Development build only
go run cmd/scripts/main.go

# Production build
go run cmd/scripts/main.go --production
```

## Architecture

- **`src/lib/components/partials/`**: Reusable UI components (Button, DataTable, FormField, etc.)
- **`src/routes/`**: SvelteKit routing with layout inheritance
- **`src/lib/stores/`**: Reactive state management
- **WebSocket client**: Real-time deployment status updates
- **Type definitions**: Shared interfaces with backend services

## Component System

15+ production-ready components with consistent APIs:
- Form elements with validation
- Data visualization (tables, metrics, progress)
- Interactive feedback (toasts, modals, loading states)
- Layout primitives with responsive behavior

See `src/lib/components/partials/README.md` for component documentation.