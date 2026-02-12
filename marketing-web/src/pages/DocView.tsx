import { useParams, Link } from "react-router-dom";

export function DocView() {
  const params = useParams<"*">();
  const slug = params["*"] ?? "";

  return (
    <div className="container mx-auto px-4 py-16 max-w-2xl">
      <div className="alert alert-info">
        <span>
          Documentation for <code className="font-mono">{slug || "—"}</code> is
          available in the repository under <code className="font-mono">docs/</code>.
        </span>
      </div>
      <Link to="/docs" className="btn btn-ghost mt-4">
        ← Back to Docs index
      </Link>
    </div>
  );
}
