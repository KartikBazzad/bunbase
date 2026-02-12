import { Link } from "react-router-dom";

const docSections = [
  {
    title: "Getting started",
    description: "Create an account and deploy your first function.",
    path: "users/getting-started",
  },
  {
    title: "Writing functions",
    description: "How to write effective JavaScript/TypeScript functions.",
    path: "users/writing-functions",
  },
  {
    title: "CLI guide",
    description: "Command-line interface reference.",
    path: "users/cli-guide",
  },
  {
    title: "Platform API reference",
    description: "REST API documentation for programmatic access.",
    path: "users/api-reference",
  },
  {
    title: "Projects guide",
    description: "Managing projects and organizing functions.",
    path: "users/projects",
  },
  {
    title: "Troubleshooting",
    description: "Common issues and solutions.",
    path: "users/troubleshooting",
  },
  {
    title: "Architecture",
    description: "High-level system architecture and components.",
    path: "architecture",
  },
  {
    title: "API paths",
    description: "Canonical API path conventions.",
    path: "api-paths",
  },
];

export function Docs() {
  return (
    <div className="container mx-auto px-4 py-16 max-w-4xl">
      <h1 className="text-4xl font-bold mb-2">Documentation</h1>
      <p className="text-lg opacity-90 mb-10">
        Guides and reference for the BunBase platform. Documentation lives in the
        repository; below are the main topics.
      </p>
      <div className="grid gap-4">
        {docSections.map((section) => (
          <Link
            key={section.path}
            to={`/docs/${section.path}`}
            className="card card-compact bg-base-200 shadow hover:shadow-md transition-shadow"
          >
            <div className="card-body">
              <h2 className="card-title text-lg">{section.title}</h2>
              <p className="opacity-90">{section.description}</p>
            </div>
          </Link>
        ))}
      </div>
      <p className="mt-8 text-sm opacity-75">
        Full documentation index and source are in the{" "}
        <code className="bg-base-200 px-1 rounded">docs/</code> directory of the
        BunBase repository.
      </p>
    </div>
  );
}
