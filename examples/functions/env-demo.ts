/**
 * Environment Variables Demo
 * Demonstrates using environment variables in functions
 */

export async function handler(req: Request): Promise<Response> {
  // Access environment variables
  const envVars = {
    // These would be set via the function's environment configuration
    apiKey: process.env.API_KEY ? "***hidden***" : "not set",
    nodeEnv: process.env.NODE_ENV || "not set",
    customVar: process.env.CUSTOM_VAR || "not set",
  };

  return Response.json({
    message: "Environment variables demo",
    environment: envVars,
    note: "Set environment variables via the /functions/:id/env endpoint",
  });
}
