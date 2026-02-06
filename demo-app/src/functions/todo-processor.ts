/**
 * Todo Processor Function
 * Processes todo-related operations like validation, notifications, etc.
 */

export default async function handler(req: Request): Promise<Response> {
  const url = new URL(req.url);
  const method = req.method;
  
  // Parse request body if present
  let body: any = null;
  if (method !== "GET" && method !== "HEAD") {
    try {
      const text = await req.text();
      if (text) {
        body = JSON.parse(text);
      }
    } catch (e) {
      // Ignore parse errors, body stays null
    }
  }
  
  // Handle different operations
  const operation = url.searchParams.get("op") || "process";
  
  switch (operation) {
    case "validate":
      // Validate todo data
      if (!body || !body.title) {
        return Response.json(
          { error: "Title is required" },
          { status: 400 }
        );
      }
      return Response.json({
        valid: true,
        message: "Todo is valid",
        timestamp: new Date().toISOString(),
      });
      
    case "notify":
      // Simulate notification processing
      return Response.json({
        notified: true,
        message: `Notification sent for todo: ${body?.title || "unknown"}`,
        timestamp: new Date().toISOString(),
      });
      
    case "process":
    default:
      // Default: process todo data
      return Response.json({
        processed: true,
        method,
        path: url.pathname,
        operation,
        data: body,
        timestamp: new Date().toISOString(),
        message: "Todo processed successfully",
      });
  }
}
