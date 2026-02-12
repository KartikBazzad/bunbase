import { Link } from "react-router-dom";
import { getConsoleUrl } from "@/lib/config";

const features = [
  {
    title: "Documents",
    description: "Document store with queries, indexes, and real-time subscriptions.",
    icon: "📄",
  },
  {
    title: "Key-value store",
    description: "Low-latency KV with optional TTL. Ideal for cache and session data.",
    icon: "🔑",
  },
  {
    title: "Functions",
    description: "Deploy and run serverless JavaScript/TypeScript in seconds.",
    icon: "⚡",
  },
  {
    title: "Real-time",
    description: "Live subscriptions and updates so your app stays in sync.",
    icon: "📡",
  },
];

export function HomePage() {
  const consoleUrl = getConsoleUrl();
  return (
    <>
      <section className="hero min-h-[80vh] bg-base-100 py-20">
        <div className="hero-content text-center">
          <div className="max-w-3xl">
            <h1 className="text-5xl font-bold">Build faster with BunBase</h1>
            <p className="py-6 text-lg opacity-90">
              One backend for documents, key-value store, functions, and
              real-time. Deploy in minutes and scale with your app—no ops
              required.
            </p>
            <div className="flex flex-wrap gap-4 justify-center">
              <a
                href={consoleUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="btn btn-primary btn-lg"
              >
                Get started
              </a>
              <Link to="/docs" className="btn btn-outline btn-lg">
                Documentation
              </Link>
            </div>
          </div>
        </div>
      </section>

      <section className="container mx-auto px-4 py-16 max-w-6xl">
        <h2 className="text-3xl font-bold text-center mb-12">
          Everything you need to ship
        </h2>
        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
          {features.map((f) => (
            <div
              key={f.title}
              className="card bg-base-200 shadow hover:shadow-md transition-shadow"
            >
              <div className="card-body">
                <span className="text-3xl mb-2" aria-hidden>
                  {f.icon}
                </span>
                <h3 className="card-title text-lg">{f.title}</h3>
                <p className="opacity-90">{f.description}</p>
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className="bg-base-200 py-20">
        <div className="container mx-auto px-4 max-w-3xl text-center">
          <h2 className="text-3xl font-bold mb-4">Ready to build?</h2>
          <p className="text-lg opacity-90 mb-8">
            Create a project, deploy your first function, and go live in
            minutes.
          </p>
          <a
            href={consoleUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="btn btn-primary btn-lg"
          >
            Open the console
          </a>
        </div>
      </section>
    </>
  );
}
