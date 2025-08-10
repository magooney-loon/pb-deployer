# Reusable Partial Components

This directory contains reusable UI components that can be used throughout the PB Deployer application. These components follow Svelte 5 patterns and provide consistent styling and behavior.

## Components

### FormField

A flexible form input component that handles various input types with consistent styling and validation.

```svelte
<script>
  import { FormField } from '$lib/components/partials';
  
  let name = '';
  let email = '';
  let age = 0;
  let server = '';
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
  id="terms"
  label="I agree to the terms and conditions"
  type="checkbox"
  bind:checked={agreedToTerms}
/>
```

**Props:**
- `id: string` - Input ID (required)
- `label: string` - Field label (required)
- `type?: 'text' | 'email' | 'password' | 'number' | 'tel' | 'url' | 'search' | 'select' | 'checkbox'` - Input type (default: "text")
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
- `class?: string` - Additional CSS classes
- `inputClass?: string` - CSS classes for input element
- `labelClass?: string` - CSS classes for label element

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

### ErrorAlert

A flexible alert component for displaying error, warning, info, or success messages.

```svelte
<script>
  import { ErrorAlert } from '$lib/components/partials';
</script>

<ErrorAlert 
  message="Something went wrong!"
  type="error"
  onDismiss={() => console.log('dismissed')}
/>

<!-- Different types -->
<ErrorAlert message="Success!" type="success" />
<ErrorAlert message="Warning!" type="warning" />
<ErrorAlert message="Info message" type="info" />
```

**Props:**
- `message: string` - The message to display
- `title?: string` - Alert title (default: "Error")
- `type?: 'error' | 'warning' | 'info' | 'success'` - Alert type (default: "error")
- `icon?: string` - Custom icon (uses defaults based on type)
- `dismissible?: boolean` - Show dismiss button (default: true)
- `onDismiss?: () => void` - Callback when dismissed
- `class?: string` - Additional CSS classes

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
- `title: string` - Card title
- `value: string | number` - Metric value
- `icon?: string` - Icon to display
- `color?: 'default' | 'blue' | 'green' | 'red' | 'yellow' | 'purple'` - Color theme (default: "default")
- `size?: 'sm' | 'md' | 'lg'` - Card size (default: "md")
- `href?: string` - Make card a link
- `onclick?: () => void` - Click handler
- `class?: string` - Additional CSS classes

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
- `color?: 'blue' | 'green' | 'red' | 'yellow' | 'gray' | 'white'` - Color theme (default: "blue")
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

### StatusBadge

A badge component for displaying status with various styles.

```svelte
<script>
  import { StatusBadge } from '$lib/components/partials';
</script>

<StatusBadge status="Online" variant="success" />
<StatusBadge status="Pending" variant="warning" dot />
<StatusBadge status="Error" variant="error" size="lg" />
<StatusBadge 
  status="Custom" 
  variant="custom" 
  customColors={{ bg: 'bg-purple-100', text: 'text-purple-800' }}
/>
```

**Props:**
- `status: string` - Status text
- `variant?: 'success' | 'warning' | 'error' | 'info' | 'gray' | 'custom'` - Badge style (default: "gray")
- `size?: 'xs' | 'sm' | 'md' | 'lg'` - Badge size (default: "sm")
- `rounded?: boolean` - Rounded badge (default: true)
- `dot?: boolean` - Show status dot (default: false)
- `customColors?: { bg: string; text: string }` - Custom colors for "custom" variant
- `class?: string` - Additional CSS classes

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
- `title: string` - Card title
- `items: T[]` - Array of items to display (generic type)
- `viewAllHref?: string` - "View all" link
- `viewAllText?: string` - "View all" text (default: "View all →")
- `emptyState: EmptyState` - Empty state configuration
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

## Usage Tips

1. **Import from index**: Always import from the index file for consistency:
   ```svelte
   import { Button, Card, ErrorAlert, FormField, EmptyState, DataTable } from '$lib/components/partials';
   ```

2. **TypeScript support**: All components are fully typed with proper TypeScript interfaces.

3. **Consistent styling**: Components use Tailwind CSS classes and follow the app's design system.

4. **Dark mode**: All components support dark mode out of the box.

5. **Accessibility**: Components include proper ARIA attributes and keyboard navigation support.

6. **Snippets over slots**: Components use Svelte 5's snippet syntax for maximum flexibility.

7. **Form validation**: Use FormField's `errorText` prop for validation messages and `helperText` for guidance.

8. **Consistent empty states**: Use EmptyState component for all "no data" scenarios to maintain consistency.

9. **Table customization**: DataTable supports both automatic rendering and custom snippets for full control over row display.

## Styling

All components use Tailwind CSS classes and support the application's color palette:
- Blue (primary)
- Green (success)
- Red (error/danger)
- Yellow (warning)
- Gray (neutral)
- Purple (accent)

Components automatically handle dark mode variants and maintain consistent spacing, shadows, and border radius throughout the application.