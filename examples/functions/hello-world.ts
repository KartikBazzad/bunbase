/**
 * Hello World Function
 * Simple example function that returns a greeting
 */

export async function handler(req: Request): Promise<Response> {
  const url = new URL(req.url);
  const name = url.searchParams.get("name") || "World";

  return Response.json({
    message: `Hello, ${name}!`,
    timestamp: new Date().toISOString(),
    method: req.method,
    path: url.pathname,
  });
}
