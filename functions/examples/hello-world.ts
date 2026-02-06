/**
 * Hello World Function Example
 * Demonstrates access to BunBase.admin() project context.
 */

export default async function handler(req: Request): Promise<Response> {
  const url = new URL(req.url);
  const name = url.searchParams.get("name") || "World";

  // BunBase.admin() is injected by the worker and reads project context
  // (project ID, API key, gateway URL) from environment variables.
  let projectContext: Record<string, string | undefined> | undefined;
  try {
    const admin = (globalThis as any).BunBase?.admin?.();
    if (admin) {
      projectContext = {
        projectId: process.env.BUNBASE_PROJECT_ID,
        gatewayUrl: process.env.BUNBASE_GATEWAY_URL,
      };
    }
  } catch {
    // If admin SDK is not available for some reason, ignore gracefully.
  }

  return Response.json({
    message: `Hello, ${name}!`,
    timestamp: new Date().toISOString(),
    method: req.method,
    path: url.pathname,
    project: projectContext,
  });
}
