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
- **`src/lib/components/main/`**: Main page components (Dashboard,Servers,Apps,etc)
- **`src/lib/components/partials/`**: UI components (Button, DataTable, FormField, etc.)
- **`src/lib/components/modals/`**: Dedicated modals (inherited from partials Modal)
- **`src/lib/api`**: Route style API client factory
- **`src/routes/`**: SvelteKit routing with layout inheritance
- **Type definitions**: Shared interfaces with backend services

## Component System

15+ production-ready components with consistent APIs:
- Form elements with validation
- Data visualization (tables, metrics, progress)
- Interactive feedback (toasts, modals, loading states)
- Layout primitives with responsive behavior

See `src/lib/components/partials/README.md` for component documentation.
