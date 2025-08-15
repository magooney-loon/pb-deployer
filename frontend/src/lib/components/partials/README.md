# Reusable Partial Components

This directory contains reusable UI components that can be used throughout the PB Deployer application. These components follow Svelte 5 patterns and provide consistent styling and behavior.

## Usage Tips

1. **Import from index**: Always import from the index file for consistency:
   ```svelte
   import {
     Background, Button, Card, DataTable, EmptyState,
     FileUpload, FormField, LoadingSpinner, MetricCard,
     ProgressBar, RecentItemsCard, StatusBadge, Toast,
     WarningBanner
   } from '$lib/components/partials';
   ```

2. **TypeScript support**: All components are fully typed with proper TypeScript interfaces and generic support where appropriate.

3. **Consistent styling**: Components use Tailwind CSS classes and follow the app's design system with consistent spacing, shadows, and border radius.

4. **Dark mode**: All components support dark mode out of the box with proper color variants.

5. **Accessibility**: Components include proper ARIA attributes, keyboard navigation support, and semantic HTML.

6. **Snippets over slots**: Components use Svelte 5's snippet syntax for maximum flexibility and type safety.

7. **Form validation**: Use FormField's `errorText` prop for validation messages and `helperText` for guidance.

8. **File handling**: FileUpload component supports single/multiple files, directories, drag-and-drop, and client-side validation.

9. **Consistent empty states**: Use EmptyState component for all "no data" scenarios to maintain consistency.

10. **Table customization**: DataTable supports both automatic rendering and custom snippets for full control over row display.

11. **Status management**: Use StatusBadge with helper functions for consistent status display across the application.

12. **User feedback**: Use Toast for temporary notifications and WarningBanner for persistent important messages.

## Styling

All components use Tailwind CSS classes and support the application's color palette:
- Blue (primary)
- Green (success)
- Red (error/danger)
- Yellow (warning)
- Gray (neutral)
- Purple (accent)
- White (contrast)

All animations respect user preferences for reduced motion when `prefers-reduced-motion: reduce` is set.

## Components

### Background

A background component with animated elements and different visual variants.

```svelte
<script>
  import { Background } from '$lib/components/partials';
</script>

<Background />
<Background variant="splash" intensity="strong" />
<Background variant="lockscreen" intensity="subtle" />
```

**Props:**
- `variant?: 'default' | 'splash' | 'lockscreen'` - Visual variant (default: "default")
- `intensity?: 'subtle' | 'medium' | 'strong'` - Background intensity (default: "medium")

### Button

A comprehensive button component with multiple variants and states.

```svelte
<script>
  import { Button } from '$lib/components/partials';
</script>

<Button>Default Button</Button>
<Button variant="outline" color="green">Outline Button</Button>
<Button variant="ghost" icon="🔄">Ghost with Icon</Button>
<Button href="/somewhere" icon="🔗">Link Button</Button>
<Button loading={true}>Loading...</Button>
<Button disabled>Disabled</Button>
```

**Props:**
- `variant?: 'primary' | 'secondary' | 'outline' | 'ghost' | 'link'` - Button style (default: "primary")
- `color?: 'blue' | 'green' | 'red' | 'yellow' | 'gray' | 'white' | 'purple'` - Color theme (default: "blue")
- `size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl'` - Button size (default: "md")
- `disabled?: boolean` - Disabled state (default: false)
- `loading?: boolean` - Loading state (default: false)
- `href?: string` - Make button a link
- `target?: string` - Link target
- `icon?: string` - Icon to display
- `iconPosition?: 'left' | 'right'` - Icon position (default: "left")
- `fullWidth?: boolean` - Full width button (default: false)
- `onclick?: () => void` - Click handler
- `type?: 'button' | 'submit' | 'reset'` - Button type (default: "button")
- `class?: string` - Additional CSS classes
- `children?: Snippet` - Button content

### Card

A flexible card container with optional header and interactive states.

```svelte
<script>
  import { Card } from '$lib/components/partials';
</script>

<Card title="Simple Card">
  <p>Card content goes here</p>
</Card>

<Card
  title="Interactive Card"
  subtitle="Click me"
  hover
  onclick={() => console.log('clicked')}
>
  <p>This card is clickable</p>
</Card>

<Card href="/link" title="Link Card">
  <p>This card is a link</p>
</Card>
```

**Props:**
- `title?: string` - Card title
- `subtitle?: string` - Card subtitle
- `padding?: 'none' | 'sm' | 'md' | 'lg' | 'xl'` - Internal padding (default: "md")
- `shadow?: 'none' | 'sm' | 'md' | 'lg' | 'xl'` - Shadow size (default: "md")
- `rounded?: 'none' | 'sm' | 'md' | 'lg' | 'xl' | 'full'` - Border radius (default: "lg")
- `hover?: boolean` - Hover effects (default: false)
- `clickable?: boolean` - Make clickable (default: false)
- `href?: string` - Make card a link
- `target?: string` - Link target
- `onclick?: () => void` - Click handler
- `class?: string` - Additional CSS classes
- `headerClass?: string` - Header CSS classes
- `bodyClass?: string` - Body CSS classes
- `children?: Snippet` - Card content

### DataTable

A comprehensive table component for displaying structured data with sorting, actions, and empty states.

```svelte
<script>
  import { DataTable } from '$lib/components/partials';

  const users = [
    { id: 1, name: 'John Doe', email: 'john@example.com', role: 'Admin' },
    { id: 2, name: 'Jane Smith', email: 'jane@example.com', role: 'User' }
  ];

  const columns = [
    { key: 'name', label: 'Name', sortable: true },
    { key: 'email', label: 'Email' },
    { key: 'role', label: 'Role', align: 'center' }
  ];
</script>

<DataTable
  data={users}
  {columns}
  striped
  hoverable
  emptyState={{
    icon: '👥',
    title: 'No users found',
    description: 'Add your first user to get started',
    primaryAction: {
      text: 'Add User',
      onclick: () => console.log('add user')
    }
  }}
>
  {#snippet children(user, index)}
    <td class="px-6 py-4">{user.name}</td>
    <td class="px-6 py-4">{user.email}</td>
    <td class="px-6 py-4 text-center">{user.role}</td>
  {/snippet}
  {#snippet actions(user, index)}
    <button onclick={() => editUser(user.id)}>Edit</button>
    <button onclick={() => deleteUser(user.id)}>Delete</button>
  {/snippet}
</DataTable>
```

**Props:**
- `data?: T[]` - Array of data objects (default: [])
- `columns: Column[]` - Column definitions (required)
- `loading?: boolean` - Loading state (default: false)
- `emptyState?: object` - Empty state configuration
  - `icon?: string` - Empty state icon
  - `title: string` - Empty state title
  - `description?: string` - Empty state description
  - `primaryAction?: object` - Primary action for empty state
- `striped?: boolean` - Alternating row colors (default: false)
- `hoverable?: boolean` - Hover effects (default: true)
- `compact?: boolean` - Reduced padding (default: false)
- `class?: string` - Additional CSS classes
- `tableClass?: string` - Table CSS classes
- `headerClass?: string` - Header CSS classes
- `bodyClass?: string` - Body CSS classes
- `rowClass?: string` - Row CSS classes
- `cellClass?: string` - Cell CSS classes
- `children?: Snippet<[T, number]>` - Custom row renderer
- `actions?: Snippet<[T, number]>` - Actions column renderer

**Column interface:**
```typescript
interface Column {
  key: string;           // Data property key
  label: string;         // Column header
  sortable?: boolean;    // Enable sorting
  width?: string;        // Column width
  align?: 'left' | 'center' | 'right';  // Text alignment
  class?: string;        // Additional CSS classes
}
```

### EmptyState

A component for displaying empty states with optional icons and call-to-action buttons.

```svelte
<script>
  import { EmptyState } from '$lib/components/partials';
</script>

<EmptyState
  icon="📁"
  title="No files found"
  description="Upload your first file to get started"
  primaryAction={{
    text: 'Upload File',
    onclick: () => console.log('upload clicked')
  }}
  secondaryText="Supported formats: PDF, DOC, TXT"
/>

<!-- Different sizes -->
<EmptyState
  title="No data"
  size="sm"
/>

<EmptyState
  icon="🚀"
  title="Ready to deploy?"
  size="lg"
  primaryAction={{
    text: 'Start Deployment',
    href: '/deploy',
    variant: 'primary',
    color: 'green'
  }}
/>
```

**Props:**
- `title: string` - Main title (required)
- `icon?: string` - Icon to display
- `description?: string` - Description text
- `primaryAction?: object` - Primary action button configuration
  - `text: string` - Button text
  - `onclick?: () => void` - Click handler
  - `href?: string` - Link URL
  - `variant?: 'primary' | 'secondary' | 'outline' | 'ghost' | 'link'` - Button variant
  - `color?: 'blue' | 'green' | 'red' | 'yellow' | 'gray' | 'white' | 'purple'` - Button color
- `secondaryText?: string` - Additional helper text
- `size?: 'sm' | 'md' | 'lg'` - Component size (default: "md")
- `class?: string` - Additional CSS classes

### FileUpload

A drag-and-drop file upload component with validation and preview.

```svelte
<script>
  import { FileUpload } from '$lib/components/partials';

  let selectedFile = null;
  let uploadError = '';

  function handleFileSelect(file) {
    selectedFile = file;
    uploadError = '';
  }

  function handleError(error) {
    uploadError = error;
  }
</script>

<FileUpload
  id="file-upload"
  label="Upload File"
  accept=".pdf,.doc,.docx"
  maxSize={10 * 1024 * 1024}
  value={selectedFile}
  errorText={uploadError}
  onFileSelect={handleFileSelect}
  onError={handleError}
  helperText="Drag and drop or click to upload"
/>

<!-- Multiple files -->
<FileUpload
  id="multi-upload"
  label="Upload Multiple Files"
  multiple
  value={selectedFiles}
  onFileSelect={handleMultipleFiles}
/>

<!-- Directory upload -->
<FileUpload
  id="dir-upload"
  label="Upload Directory"
  directory
  value={selectedDirectory}
  onFileSelect={handleDirectorySelect}
/>
```

**Props:**
- `id: string` - Input ID (required)
- `label: string` - Field label (required)
- `accept?: string` - Accepted file types (default: "")
- `multiple?: boolean` - Allow multiple files (default: false)
- `directory?: boolean` - Allow directory upload (default: false)
- `maxSize?: number` - Maximum file size in bytes (default: 50MB)
- `required?: boolean` - Required field (default: false)
- `disabled?: boolean` - Disabled state (default: false)
- `helperText?: string` - Helper text
- `errorText?: string` - Error message
- `value?: File | File[] | null` - Selected file(s) (bindable)
- `class?: string` - Additional CSS classes
- `onFileSelect?: (files: File | File[] | null) => void` - File selection callback
- `onError?: (error: string) => void` - Error callback

### FormField

A flexible form input component that handles various input types with consistent styling and validation.

```svelte
<script>
  import { FormField } from '$lib/components/partials';

  let name = '';
  let email = '';
  let age = 0;
  let server = '';
  let description = '';
  let agreedToTerms = false;
</script>

<FormField
  id="name"
  label="Full Name"
  bind:value={name}
  placeholder="Enter your name"
  required
/>

<FormField
  id="email"
  label="Email Address"
  type="email"
  bind:value={email}
  helperText="We'll never share your email"
  required
/>

<FormField
  id="age"
  label="Age"
  type="number"
  bind:value={age}
  min={18}
  max={120}
/>

<FormField
  id="server"
  label="Select Server"
  type="select"
  bind:value={server}
  placeholder="Choose a server"
  options={[
    { value: 'prod', label: 'Production' },
    { value: 'staging', label: 'Staging' },
    { value: 'dev', label: 'Development' }
  ]}
/>

<FormField
  id="description"
  label="Description"
  type="textarea"
  bind:value={description}
  rows={5}
  placeholder="Enter description..."
/>

<FormField
  id="terms"
  label="I agree to the terms and conditions"
  type="checkbox"
  bind:checked={agreedToTerms}
/>
```

**Props:**
- `id: string` - Input ID (required)
- `label: string` - Field label (required)
- `type?: 'text' | 'email' | 'password' | 'number' | 'tel' | 'url' | 'search' | 'select' | 'checkbox' | 'textarea'` - Input type (default: "text")
- `placeholder?: string` - Placeholder text
- `required?: boolean` - Required field (default: false)
- `disabled?: boolean` - Disabled state (default: false)
- `readonly?: boolean` - Readonly state (default: false)
- `helperText?: string` - Helper text below input
- `errorText?: string` - Error message (shows in red)
- `value?: string | number` - Input value (bindable)
- `checked?: boolean` - Checkbox state (bindable)
- `options?: Array<{value: string | number, label: string, disabled?: boolean}>` - Options for select type
- `min?: number` - Minimum value for number inputs
- `max?: number` - Maximum value for number inputs
- `step?: number | string` - Step value for number inputs
- `rows?: number` - Rows for textarea (default: 3)
- `class?: string` - Additional CSS classes
- `oninput?: (event: Event) => void` - Input event handler
- `onchange?: (event: Event) => void` - Change event handler

### LoadingSpinner

A customizable loading spinner with text.

```svelte
<script>
  import { LoadingSpinner } from '$lib/components/partials';
</script>

<LoadingSpinner />
<LoadingSpinner text="Loading data..." size="lg" color="green" />
<LoadingSpinner centered={false} />
```

**Props:**
- `text?: string` - Loading text (default: "Loading...")
- `size?: 'sm' | 'md' | 'lg'` - Spinner size (default: "md")
- `color?: 'blue' | 'gray' | 'green' | 'red' | 'yellow'` - Spinner color (default: "blue")
- `centered?: boolean` - Center the spinner (default: true)
- `class?: string` - Additional CSS classes

### MetricCard

A card component for displaying metrics with optional icons and colors.

```svelte
<script>
  import { MetricCard } from '$lib/components/partials';
</script>

<MetricCard title="Total Users" value={1234} icon="👥" />
<MetricCard
  title="Revenue"
  value="$12,345"
  icon="💰"
  color="green"
  size="lg"
  onclick={() => console.log('clicked')}
/>
<MetricCard title="Visits" value={567} href="/analytics" />
```

**Props:**
- `title: string` - Card title (required)
- `value: string | number` - Metric value (required)
- `icon?: string` - Icon to display
- `color?: 'default' | 'blue' | 'green' | 'red' | 'yellow' | 'purple'` - Color theme (default: "default")
- `size?: 'sm' | 'md' | 'lg'` - Card size (default: "md")
- `href?: string` - Make card a link
- `onclick?: () => void` - Click handler
- `class?: string` - Additional CSS classes

### ProgressBar

A progress bar component with customizable styling and animations.

```svelte
<script>
  import { ProgressBar } from '$lib/components/partials';
</script>

<ProgressBar value={75} label="Upload Progress" />
<ProgressBar value={50} max={200} showPercentage={false} />
<ProgressBar
  value={90}
  color="green"
  size="lg"
  striped
  animated
  label="Processing..."
/>
```

**Props:**
- `value?: number` - Current progress value (default: 0)
- `max?: number` - Maximum value (default: 100)
- `label?: string` - Progress label
- `showPercentage?: boolean` - Show percentage text (default: true)
- `color?: 'blue' | 'green' | 'yellow' | 'red' | 'gray'` - Progress bar color (default: "blue")
- `size?: 'sm' | 'md' | 'lg'` - Bar size (default: "md")
- `animated?: boolean` - Smooth animations (default: true)
- `striped?: boolean` - Striped pattern (default: false)
- `class?: string` - Additional CSS classes

### RecentItemsCard

A specialized card for displaying lists of recent items with empty states.

```svelte
<script>
  import { RecentItemsCard } from '$lib/components/partials';

  const servers = [
    { id: 1, name: 'Server 1', status: 'online' },
    { id: 2, name: 'Server 2', status: 'offline' }
  ];
</script>

<RecentItemsCard
  title="Recent Servers"
  items={servers}
  viewAllHref="/servers"
  emptyState={{
    message: 'No servers yet',
    ctaText: 'Add your first server →',
    ctaHref: '/servers/new'
  }}
>
  {#snippet children(server, index)}
    <div class="flex-1">
      <h4>{server.name}</h4>
      <p>Status: {server.status}</p>
    </div>
  {/snippet}
</RecentItemsCard>
```

**Props:**
- `title: string` - Card title (required)
- `items: T[]` - Array of items to display (generic type) (required)
- `viewAllHref?: string` - "View all" link
- `viewAllText?: string` - "View all" text (default: "View all →")
- `emptyState: EmptyState` - Empty state configuration (required)
- `itemClass?: string` - CSS classes for item containers
- `class?: string` - Additional CSS classes
- `children?: Snippet<[T, number]>` - Item renderer snippet

**EmptyState interface:**
```typescript
interface EmptyState {
  message: string;           // Main empty message
  ctaText?: string;         // Call-to-action text
  ctaHref?: string;         // Call-to-action link
  secondaryText?: string;   // Additional helper text
}
```

### StatusBadge

A badge component for displaying status with various styles and helper functions.

```svelte
<script>
  import { StatusBadge, getServerStatusBadge, getAppStatusBadge } from '$lib/components/partials';

  // Manual usage
  <StatusBadge status="Online" variant="success" />
  <StatusBadge status="Pending" variant="warning" dot />
  <StatusBadge status="Error" variant="error" size="lg" />

  // Using helper functions
  const serverBadge = getServerStatusBadge(server);
  <StatusBadge status={serverBadge.text} variant={serverBadge.variant} />

  const appBadge = getAppStatusBadge(app);
  <StatusBadge status={appBadge.text} variant={appBadge.variant} />
</script>

<StatusBadge
  status="Custom"
  variant="custom"
  customColors={{ bg: 'bg-purple-100', text: 'text-purple-800' }}
/>
```

**Props:**
- `status: string` - Status text (required)
- `variant?: 'success' | 'warning' | 'error' | 'info' | 'gray' | 'custom'` - Badge style (default: "gray")
- `size?: 'xs' | 'sm' | 'md' | 'lg'` - Badge size (default: "sm")
- `rounded?: boolean` - Rounded badge (default: true)
- `dot?: boolean` - Show status dot (default: false)
- `customColors?: { bg: string; text: string }` - Custom colors for "custom" variant
- `class?: string` - Additional CSS classes

**Helper Functions:**
- `getServerStatusBadge(server: Server): StatusBadgeResult` - Get badge config for server status
- `getAppStatusBadge(app: App): StatusBadgeResult` - Get badge config for app status
- `getAppStatusIcon(status: string): string` - Get icon for app status
- `formatTimestamp(timestamp: string): string` - Format timestamp string

### Toast

A toast notification component for displaying temporary messages.

```svelte
<script>
  import { Toast } from '$lib/components/partials';
</script>

<Toast message="Operation completed successfully!" type="success" />
<Toast
  message="Something went wrong!"
  type="error"
  onDismiss={() => console.log('dismissed')}
/>
<Toast message="Warning message" type="warning" icon="⚠️" />
<Toast message="Info message" type="info" dismissible={false} />
```

**Props:**
- `message: string` - Toast message (required)
- `type?: 'error' | 'warning' | 'info' | 'success'` - Toast type (default: "error")
- `icon?: string` - Custom icon (uses defaults based on type)
- `dismissible?: boolean` - Show dismiss button (default: true)
- `onDismiss?: () => void` - Callback when dismissed
- `class?: string` - Additional CSS classes

### WarningBanner

A banner component for displaying important warnings at the top of the page.

```svelte
<script>
  import { WarningBanner } from '$lib/components/partials';
</script>

<WarningBanner />
<WarningBanner
  message="Custom warning message"
  color="red"
  size="sm"
  onDismiss={() => console.log('banner dismissed')}
/>
<WarningBanner
  message="Info banner"
  icon="ℹ️"
  color="blue"
  dismissible={false}
/>
```

**Props:**
- `message?: string` - Warning message (default: "Always close this application using Ctrl+C to prevent data loss and ensure proper cleanup.")
- `icon?: string` - Warning icon (default: "⚠️")
- `dismissible?: boolean` - Allow dismissing (default: true)
- `color?: 'yellow' | 'blue' | 'red' | 'gray'` - Banner color (default: "yellow")
- `size?: 'xs' | 'sm'` - Banner size (default: "sm")
- `class?: string` - Additional CSS classes
- `onDismiss?: () => void` - Dismiss callback
