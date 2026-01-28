/**
 * QuickJS-NG Hello World Function Example
 * 
 * This function is designed to run in QuickJS-NG runtime.
 * It uses standard Web APIs (Request/Response) that are available in QuickJS.
 */

export default async function handler(req: Request): Promise<Response> {
  const url = new URL(req.url);
  const name = url.searchParams.get("name") || "World";
  
  // QuickJS-NG supports modern JavaScript features
  const timestamp = new Date().toISOString();
  const method = req.method;
  const path = url.pathname;
  
  return Response.json({
    message: `Hello, ${name}!`,
    timestamp,
    method,
    path,
    runtime: "quickjs-ng",
    userAgent: req.headers.get("user-agent") || "unknown",
  });
}
