# BunBase Authentication Example

A comprehensive React-based authentication example application demonstrating all authentication features of the BunBase JavaScript SDK.

## Features

This example demonstrates:

- **User Registration** - Sign up with email, password, and name
- **User Login** - Sign in with email and password
- **Session Management** - Persistent sessions with automatic token refresh
- **User Profile** - View current user information and account details
- **Password Reset** - Complete forgot password flow with email verification
- **Email Verification** - Verify email addresses with secure tokens
- **Sign Out** - Securely sign out and clear session
- **Protected Routes** - Dashboard only accessible when authenticated
- **Error Handling** - User-friendly error messages and validation
- **Loading States** - Proper loading indicators during async operations

## Prerequisites

- [Bun](https://bun.sh) runtime installed (v1.0+)
- A running BunBase server (default: `http://localhost:3000/api`)
- A BunBase API key (optional - defaults to example key)

## Setup

1. **Install dependencies:**

```bash
bun install
```

2. **Configure environment variables (optional):**

Copy `.env.example` to `.env` and update with your values:

```bash
cp .env.example .env
```

Edit `.env`:
```
BUNBASE_API_KEY=your_api_key_here
BUNBASE_BASE_URL=http://localhost:3000/api
```

If you don't create a `.env` file, the app will use default values.

3. **Start the development server:**

```bash
bun dev
```

The application will be available at `http://localhost:3000` (or the port shown in the console).

## Usage

### Sign Up

1. Navigate to the sign up page (or click "Sign up" link)
2. Enter your name, email, and password
3. Click "Sign Up"
4. You'll be automatically signed in after successful registration
5. Check your email for a verification link

### Sign In

1. Enter your email and password
2. Click "Sign In"
3. You'll be redirected to the dashboard upon successful authentication

### Forgot Password

1. Click "Forgot password?" on the sign in page
2. Enter your email address
3. Check your email for a password reset link
4. Click the link and enter your new password

### Email Verification

1. After signing up, check your email for a verification link
2. Click the link to verify your email address
3. Your account status will be updated automatically

### Dashboard

Once authenticated, you can:

- View your profile information (name, email, verification status)
- See account creation and update dates
- Sign out securely

## Project Structure

```
src/
├── lib/
│   ├── auth-context.tsx    # Authentication state management
│   └── client.ts           # BunBase SDK client initialization
├── components/
│   ├── SignUp.tsx          # User registration form
│   ├── SignIn.tsx          # User login form
│   ├── ForgotPassword.tsx  # Password reset request
│   ├── ResetPassword.tsx   # Password reset form
│   ├── VerifyEmail.tsx     # Email verification handler
│   └── Dashboard.tsx        # User profile dashboard
├── App.tsx                 # Main app with routing
├── frontend.tsx            # React entry point
└── index.css               # Application styles
```

## API Integration

This example uses the BunBase JavaScript SDK (`@bunbase/js-sdk`) which provides:

- `client.auth.signUp(email, password, name)` - Register new user
- `client.auth.signIn(email, password)` - Sign in user
- `client.auth.signOut()` - Sign out user
- `client.auth.getUser()` - Get current user
- `client.auth.verifyEmail(token)` - Verify email address
- `client.auth.forgotPassword(email)` - Request password reset
- `client.auth.resetPassword(token, password)` - Reset password

## Session Management

Sessions are automatically persisted in `localStorage` and restored on page load. The authentication context:

- Automatically loads saved sessions on mount
- Validates sessions by fetching current user
- Clears invalid/expired sessions
- Provides loading and error states

## Development

### Development Mode

```bash
bun dev
```

Features:
- Hot module reloading (HMR)
- Browser console logs echoed to server
- Automatic rebuilds on file changes

### Production Build

```bash
bun run build
```

This creates an optimized production build in the `dist/` directory.

### Production Server

```bash
bun start
```

Runs the production build.

## Troubleshooting

### "Failed to sign in" errors

- Ensure the BunBase server is running at the configured `BUNBASE_BASE_URL`
- Verify your API key is correct
- Check that the email/password combination is valid

### Session not persisting

- Check browser console for localStorage errors
- Ensure cookies/localStorage are enabled in your browser
- Try clearing browser storage and signing in again

### Email verification not working

- Check that email service is configured on the BunBase server
- Verify the token in the URL is valid and not expired
- Check server logs for email sending errors

## Learn More

- [BunBase Documentation](https://docs.bunbase.com)
- [BunBase JavaScript SDK](../packages/js-sdk/README.md)
- [Bun Runtime](https://bun.sh/docs)

## License

This example is part of the BunBase project.
