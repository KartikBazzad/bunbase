# Design System Documentation

This document describes the design system and UI components available in the BunBase Platform.

## Color Palette

### Primary Colors
- **Primary Blue**: Used for main actions, links, and brand elements
  - `primary-500`: Main primary color (#0ea5e9)
  - `primary-600`: Hover states (#0284c7)
  - `primary-700`: Active/pressed states (#0369a1)

### Accent Colors
- **Purple**: Used for secondary actions and highlights
  - `accent-500`: Main accent color (#a855f7)

### Semantic Colors
- **Success**: Green tones for success states
- **Warning**: Yellow/amber tones for warnings
- **Error**: Red tones for errors
- **Gray**: Neutral tones for text, borders, backgrounds

## Typography

- **Font Family**: Inter (sans-serif), JetBrains Mono (monospace)
- **Headings**: Use semantic HTML (`h1`, `h2`, etc.) with Tailwind text size classes
- **Body**: Default text uses `text-gray-900` on `bg-gray-50`

## Components

### Buttons

```tsx
// Primary button
<button className="btn-primary">Click me</button>

// Secondary button
<button className="btn-secondary">Cancel</button>

// Outline button
<button className="btn-outline">Learn more</button>

// Danger button
<button className="btn-danger">Delete</button>

// Ghost button
<button className="btn-ghost">Skip</button>

// Sizes
<button className="btn-primary btn-sm">Small</button>
<button className="btn-primary btn-lg">Large</button>
```

### Cards

```tsx
<div className="card">
  <div className="card-header">
    <h2>Card Title</h2>
  </div>
  <div className="card-body">
    <p>Card content goes here</p>
  </div>
  <div className="card-footer">
    <button className="btn-primary">Action</button>
  </div>
</div>
```

### Badges

```tsx
<span className="badge-primary">Primary</span>
<span className="badge-success">Success</span>
<span className="badge-warning">Warning</span>
<span className="badge-error">Error</span>
<span className="badge-gray">Gray</span>
```

### Inputs

```tsx
// Standard input
<input type="text" className="input" placeholder="Enter text" />

// Error state
<input type="text" className="input input-error" placeholder="Error state" />
```

### Links

```tsx
<a href="#" className="link">Click here</a>
```

### Loading Spinner

```tsx
<div className="spinner"></div>
```

## Layout Utilities

### Container

```tsx
<div className="container-custom">
  {/* Content with max-width and responsive padding */}
</div>
```

## Usage Examples

### Login Form

```tsx
<div className="card max-w-md mx-auto">
  <div className="card-header">
    <h1 className="text-2xl font-bold">Sign In</h1>
  </div>
  <div className="card-body space-y-4">
    <div>
      <label className="block text-sm font-medium text-gray-700 mb-1">
        Email
      </label>
      <input type="email" className="input" placeholder="you@example.com" />
    </div>
    <div>
      <label className="block text-sm font-medium text-gray-700 mb-1">
        Password
      </label>
      <input type="password" className="input" />
    </div>
    <button className="btn-primary w-full">Sign In</button>
  </div>
</div>
```

### Project Card

```tsx
<div className="card hover:shadow-medium transition-shadow cursor-pointer">
  <div className="card-body">
    <div className="flex items-start justify-between">
      <div>
        <h3 className="text-lg font-semibold mb-1">My Project</h3>
        <p className="text-sm text-gray-600">3 functions deployed</p>
      </div>
      <span className="badge-success">Active</span>
    </div>
  </div>
</div>
```

## Best Practices

1. **Consistency**: Always use the predefined component classes
2. **Spacing**: Use Tailwind spacing utilities (`space-y-4`, `gap-4`, etc.)
3. **Responsive**: Use Tailwind responsive prefixes (`sm:`, `md:`, `lg:`) for mobile-first design
4. **Accessibility**: Include proper labels, ARIA attributes, and keyboard navigation
5. **States**: Always provide hover, focus, and disabled states for interactive elements

## Customization

To customize colors, fonts, or other design tokens, edit `tailwind.config.js` and update the theme extensions.
