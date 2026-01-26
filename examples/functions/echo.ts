/**
 * Echo Function
 * Echoes back the request data with metadata
 */

export async function handler(req: Request): Promise<Response> {
  const url = new URL(req.url);
  
  // Get request body if present
  let body: any = null;
  if (req.method === "POST" || req.method === "PUT" || req.method === "PATCH") {
    try {
      const text = await req.text();
      if (text) {
        try {
          body = JSON.parse(text);
        } catch {
          body = text;
        }
      }
    } catch {
      // No body
    }
  }

  // Get query parameters
  const queryParams: Record<string, string> = {};
  url.searchParams.forEach((value, key) => {
    queryParams[key] = value;
  });

  // Get headers (filter out sensitive ones)
  const headers: Record<string, string> = {};
  req.headers.forEach((value, key) => {
    if (!key.toLowerCase().includes("authorization") && 
        !key.toLowerCase().includes("cookie")) {
      headers[key] = value;
    }
  });

  return Response.json({
    method: req.method,
    url: req.url,
    pathname: url.pathname,
    query: queryParams,
    headers,
    body,
    timestamp: new Date().toISOString(),
  });
}
