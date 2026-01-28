# BunBase Platform Web

React + Vite frontend for the BunBase Platform with Tailwind CSS design system.

## Features

- ⚡ **Vite** - Fast build tool and dev server
- ⚛️ **React 19** - Latest React with TypeScript
- 🎨 **Tailwind CSS** - Utility-first CSS framework
- 🎯 **Design System** - Pre-built component classes and color palette
- 🛣️ **React Router** - Client-side routing (ready to use)

## Getting Started

### Install Dependencies

```bash
npm install
```

### Development

```bash
npm run dev
```

The app will be available at `http://localhost:5173`

### Build

```bash
npm run build
```

### Preview Production Build

```bash
npm run preview
```

## Design System

The project includes a comprehensive design system with:

- **Color Palette**: Primary blue, accent purple, semantic colors (success, warning, error)
- **Components**: Buttons, cards, badges, inputs, links
- **Typography**: Inter font family with proper heading styles
- **Utilities**: Container, spacing, shadows

See [DESIGN_SYSTEM.md](./DESIGN_SYSTEM.md) for complete documentation.

## Project Structure

```
platform-web/
├── src/
│   ├── components/     # Reusable components
│   ├── pages/          # Page components
│   ├── hooks/          # Custom React hooks
│   ├── lib/            # Utilities and API client
│   ├── App.tsx         # Main app component
│   ├── main.tsx        # Entry point
│   └── index.css       # Tailwind directives and component styles
├── public/             # Static assets
├── tailwind.config.js  # Tailwind configuration
└── vite.config.ts      # Vite configuration
```

## Available Component Classes

### Buttons
- `btn-primary` - Primary action button
- `btn-secondary` - Secondary button
- `btn-outline` - Outlined button
- `btn-danger` - Destructive action button
- `btn-ghost` - Minimal button
- `btn-sm`, `btn-lg` - Size variants

### Cards
- `card` - Card container
- `card-header` - Card header section
- `card-body` - Card content section
- `card-footer` - Card footer section

### Badges
- `badge-primary`, `badge-success`, `badge-warning`, `badge-error`, `badge-gray`

### Inputs
- `input` - Standard input field
- `input-error` - Error state input

### Other
- `link` - Styled link
- `spinner` - Loading spinner
- `container-custom` - Responsive container

## Environment Variables

Create a `.env` file for environment-specific configuration:

```env
VITE_API_URL=http://localhost:3001/api
```

## Next Steps

1. Set up API client in `src/lib/api.ts`
2. Create authentication hooks in `src/hooks/useAuth.ts`
3. Build login/signup pages
4. Create dashboard and project management pages
5. Integrate with the Go backend API

## License

MIT
