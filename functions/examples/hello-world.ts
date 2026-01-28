/**
 * Hello World Function Example
 * Simple function that returns a greeting
 */

export default async function handler(req: Request): Promise<Response> {
  const url = new URL(req.url);
  const name = url.searchParams.get("name") || "World";
  
  return Response.json({
    message: `Hello, ${name}!`,
    timestamp: new Date().toISOString(),
    method: req.method,
    path: url.pathname,
  });
}
