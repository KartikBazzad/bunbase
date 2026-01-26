/**
 * JSON Processor Function
 * Processes JSON data and returns transformed result
 */

export async function handler(req: Request): Promise<Response> {
  if (req.method !== "POST") {
    return Response.json(
      { error: "Method not allowed. Use POST." },
      { status: 405 },
    );
  }

  try {
    const data = await req.json();

    // Process the data
    const processed = {
      received: data,
      processedAt: new Date().toISOString(),
      itemCount: Array.isArray(data) ? data.length : 1,
      keys: typeof data === "object" && data !== null ? Object.keys(data) : [],
    };

    return Response.json({
      success: true,
      result: processed,
    });
  } catch (error: any) {
    return Response.json(
      {
        success: false,
        error: "Invalid JSON",
        message: error.message,
      },
      { status: 400 },
    );
  }
}
