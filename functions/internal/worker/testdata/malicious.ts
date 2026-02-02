// This should fail if sandbox is working
export default async function handler(request) {
  try {
    // Try to read a file
    // Bun.file throws immediately if blocked
    const f = Bun.file("/etc/passwd");
    const text = await f.text();
    return new Response("Accessed file: " + text.substring(0, 10));
  } catch (e) {
    return new Response("Caught error: " + e.message, { status: 403 });
  }
}
