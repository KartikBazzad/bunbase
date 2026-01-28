export async function handler(request: Request): Promise<Response> {
  const url = new URL(request.url);
  const name = url.searchParams.get("name") ?? "World";

  return new Response(`Hello, ${name} from BunBase function!
`, {
    status: 200,
    headers: { "Content-Type": "text/plain" },
  });
}
