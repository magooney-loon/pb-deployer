# pb-deployer Frontend

Svelte/Kit 5 component-driven architecture.

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
