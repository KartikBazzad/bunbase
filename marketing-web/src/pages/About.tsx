import { Link } from "react-router-dom";
import { getConsoleUrl } from "@/lib/config";

const offerings = [
  "Document store with queries and real-time subscriptions",
  "Key-value store with optional TTL for cache and sessions",
  "Serverless functions in JavaScript/TypeScript",
  "Real-time subscriptions and live updates",
];

export function About() {
  const consoleUrl = getConsoleUrl();
  return (
    <div className="container mx-auto px-4 py-16 max-w-3xl">
      <h1 className="text-4xl font-bold mb-6">About BunBase</h1>
      <div className="prose prose-lg max-w-none">
        <p className="lead">
          BunBase is a backend platform designed for speed and simplicity. It
          brings together documents, key-value storage, serverless functions,
          and real-time capabilities in one place.
        </p>
        <p>
          Whether you're building a small project or scaling to production,
          BunBase provides the primitives you need without the operational
          overhead. Use the console to manage projects, deploy functions, and
          explore your data.
        </p>
      </div>

      <h2 className="text-2xl font-bold mt-12 mb-4">What we offer</h2>
      <ul className="list-disc list-inside space-y-2 opacity-90">
        {offerings.map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>

      <p className="mt-10">
        <a
          href={consoleUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="link link-primary"
        >
          Open the console
        </a>{" "}
        to get started, or head to the{" "}
        <Link to="/docs" className="link link-primary">
          docs
        </Link>{" "}
        for guides and API reference.
      </p>
    </div>
  );
}
