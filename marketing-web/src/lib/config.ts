/**
 * Console URL for "Open Console" / "Get started" links.
 * Set VITE_CONSOLE_URL in .env for production (e.g. https://console.bunbase.com).
 */
export function getConsoleUrl(): string {
  return import.meta.env.VITE_CONSOLE_URL ?? "http://console.localhost";
}
