import { Link } from "react-router-dom";
import { useConfig } from "@/contexts/ConfigContext";

export function Home() {
  const { isConfigured } = useConfig();

  return (
    <div className="rounded-lg border bg-white p-6 shadow-sm">
      <h1 className="mb-2 text-2xl font-semibold">BunBase Demo App</h1>
      <p className="mb-4 text-gray-600">
        This app uses the BunBase TypeScript SDK for documents, auth, and functions.
      </p>
      {!isConfigured ? (
        <p className="text-sm text-amber-700">
          Set your <Link to="/settings" className="underline">Project API key and Project ID in Settings</Link> to use Documents and Functions.
        </p>
      ) : (
        <ul className="list-inside list-disc space-y-1 text-sm">
          <li><Link to="/documents" className="text-blue-600">Documents</Link> – CRUD on a <code className="rounded bg-gray-100 px-1">tasks</code> collection via the SDK</li>
          <li><Link to="/references" className="text-blue-600">References</Link> – Cross-collection references demo: users and posts with <code className="rounded bg-gray-100 px-1">author_id</code>, 409 on invalid ref, and restrict / set_null / cascade on delete</li>
          <li><Link to="/functions" className="text-blue-600">Functions</Link> – Invoke a function by name with optional JSON body</li>
          <li><Link to="/settings" className="text-blue-600">Settings</Link> – Change API URL, token, or project</li>
        </ul>
      )}
    </div>
  );
}
