/**
 * Calculator Function
 * Simple calculator API
 */

export async function handler(req: Request): Promise<Response> {
  const url = new URL(req.url);

  if (req.method === "GET") {
    // GET: /calculator?a=10&b=5&op=add
    const a = parseFloat(url.searchParams.get("a") || "0");
    const b = parseFloat(url.searchParams.get("b") || "0");
    const op = url.searchParams.get("op") || "add";

    const result = calculate(a, b, op);

    return Response.json({
      operation: op,
      operands: { a, b },
      result,
    });
  } else if (req.method === "POST") {
    // POST: { "a": 10, "b": 5, "op": "multiply" }
    try {
      const data = await req.json();
      const { a, b, op } = data;

      if (typeof a !== "number" || typeof b !== "number") {
        return Response.json(
          { error: "Invalid operands. 'a' and 'b' must be numbers." },
          { status: 400 },
        );
      }

      const result = calculate(a, b, op || "add");

      return Response.json({
        operation: op || "add",
        operands: { a, b },
        result,
      });
    } catch (error: any) {
      return Response.json(
        { error: "Invalid JSON", message: error.message },
        { status: 400 },
      );
    }
  }

  return Response.json(
    { error: "Method not allowed" },
    { status: 405 },
  );
}

function calculate(a: number, b: number, op: string): number {
  switch (op.toLowerCase()) {
    case "add":
    case "+":
      return a + b;
    case "subtract":
    case "sub":
    case "-":
      return a - b;
    case "multiply":
    case "mul":
    case "*":
      return a * b;
    case "divide":
    case "div":
    case "/":
      if (b === 0) {
        throw new Error("Division by zero");
      }
      return a / b;
    case "power":
    case "pow":
    case "^":
      return Math.pow(a, b);
    default:
      throw new Error(`Unknown operation: ${op}`);
  }
}
