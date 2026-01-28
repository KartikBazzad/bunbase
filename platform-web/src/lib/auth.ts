/**
 * Authentication utilities
 */

export interface User {
  id: string;
  email: string;
  name: string;
  created_at: string;
  updated_at: string;
}

export function isAuthenticated(): boolean {
  // Check if session cookie exists (browser handles this automatically)
  // We'll verify with the server when needed
  return true; // Optimistic - actual check happens via API
}
