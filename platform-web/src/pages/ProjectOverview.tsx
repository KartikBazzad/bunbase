import { useParams, Link } from "react-router-dom";
import { api, type ProjectConfig } from "../lib/api";
import { ProjectServicesCard } from "../components/projects/ProjectServicesCard";
import { useEffect, useState } from "react";
import { Zap, Database as DatabaseIcon } from "lucide-react";

interface Project {
  id: string;
  name: string;
  slug: string;
  owner_id: string;
  created_at: string;
  updated_at: string;
}

export function ProjectOverview() {
  const { id } = useParams<{ id: string }>();
  const [project, setProject] = useState<Project | null>(null);
  const [projectConfig, setProjectConfig] = useState<ProjectConfig | null>(
    null
  );
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [functionsCount, setFunctionsCount] = useState<number | null>(null);
  const [collectionsCount, setCollectionsCount] = useState<number | null>(null);

  useEffect(() => {
    if (id) {
      loadProject();
      loadProjectConfig();
      loadCounts();
    }
  }, [id]);

  const loadProject = async () => {
    try {
      const data = await api.getProject(id as string);
      setProject(data as Project);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load project");
    } finally {
      setLoading(false);
    }
  };

  const loadProjectConfig = async () => {
    try {
      const data = await api.getProjectConfig(id as string);
      setProjectConfig(data);
    } catch (err) {
      console.error("Failed to load project config:", err);
    }
  };

  const loadCounts = async () => {
    if (!id) return;
    try {
      const [fns, cols] = await Promise.all([
        api.listFunctions(id),
        api.listCollections(id),
      ]);
      setFunctionsCount(Array.isArray(fns) ? fns.length : 0);
      setCollectionsCount(Array.isArray(cols) ? cols.length : 0);
    } catch {
      setFunctionsCount(0);
      setCollectionsCount(0);
    }
  };

  if (loading && !project) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <span className="loading loading-spinner loading-lg"></span>
      </div>
    );
  }

  if (error || !project) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="card max-w-md bg-base-100 shadow-xl">
          <div className="card-body text-center">
            <p className="text-error mb-4">{error || "Project not found"}</p>
            <Link to="/dashboard" className="link link-primary">
              Back to Dashboard
            </Link>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-base-200 flex flex-col">
      <main className="container mx-auto px-4 sm:px-6 lg:px-8 max-w-7xl py-8">
        <div className="space-y-6">
          {/* Project summary */}
          <div className="card bg-base-100 shadow-sm">
            <div className="card-body">
              <div className="flex flex-wrap items-start justify-between gap-4">
                <div>
                  <h2 className="text-xl font-bold">{project.name}</h2>
                  <p className="text-sm text-base-content/70 mt-1">
                    <span className="bg-base-300 text-xs px-2 py-0.5 rounded mr-2">
                      {project.slug}
                    </span>
                    Created {new Date(project.created_at).toLocaleDateString()}
                    {" · "}
                    Updated {new Date(project.updated_at).toLocaleString()}
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <code className="text-xs bg-base-300 px-2 py-1 rounded text-base-content/70">
                    {project.id}
                  </code>
                  <button
                    type="button"
                    className="btn btn-ghost btn-xs"
                    onClick={() => navigator.clipboard.writeText(project.id)}
                    aria-label="Copy project ID"
                  >
                    Copy
                  </button>
                </div>
              </div>
            </div>
          </div>

          {/* Stats row */}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            <Link
              to={`/projects/${id}/functions`}
              className="card bg-base-100 shadow-sm hover:shadow-md transition-shadow"
            >
              <div className="card-body items-center text-center">
                <Zap className="w-8 h-8 text-primary mb-1" />
                <p className="text-2xl font-bold">
                  {functionsCount === null ? "—" : functionsCount}
                </p>
                <p className="text-sm text-base-content/70">Functions</p>
              </div>
            </Link>
            <Link
              to={`/projects/${id}/database`}
              className="card bg-base-100 shadow-sm hover:shadow-md transition-shadow"
            >
              <div className="card-body items-center text-center">
                <DatabaseIcon className="w-8 h-8 text-primary mb-1" />
                <p className="text-2xl font-bold">
                  {collectionsCount === null ? "—" : collectionsCount}
                </p>
                <p className="text-sm text-base-content/70">Collections</p>
              </div>
            </Link>
            <div className="card bg-base-100 shadow-sm">
              <div className="card-body">
                <h3 className="font-semibold mb-2">Quick Actions</h3>
                <div className="space-y-2">
                  <Link
                    to={`/projects/${id}/database`}
                    className="btn btn-outline btn-block btn-sm"
                  >
                    View Database
                  </Link>
                  <Link
                    to={`/projects/${id}/functions`}
                    className="btn btn-outline btn-block btn-sm"
                  >
                    Manage Functions
                  </Link>
                  <Link
                    to={`/projects/${id}/settings`}
                    className="btn btn-outline btn-block btn-sm"
                  >
                    Settings
                  </Link>
                </div>
              </div>
            </div>
            <div className="card bg-base-100 shadow-sm">
              <div className="card-body">
                <h3 className="font-semibold mb-2">Resources</h3>
                <div className="space-y-2 text-sm">
                  <a
                    href="https://bunbase.dev/docs"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="link link-primary block"
                  >
                    Documentation
                  </a>
                  <a
                    href={`https://bunbase.dev/projects/${project.slug}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="link link-primary block"
                  >
                    Live URL
                  </a>
                  <button
                    type="button"
                    className="link link-primary block text-left"
                    onClick={() =>
                      navigator.clipboard.writeText(
                        `bunbase projects use ${project.id}`
                      )
                    }
                  >
                    Copy CLI Command
                  </button>
                </div>
              </div>
            </div>
          </div>

          {/* API endpoints (secondary, collapsible) */}
          {projectConfig && (
            <div className="collapse collapse-arrow bg-base-100 shadow-sm">
              <input type="checkbox" aria-label="Toggle API endpoints" />
              <div className="collapse-title font-semibold">API endpoints</div>
              <div className="collapse-content">
                <p className="text-sm text-base-content/70 mb-4">
                  Gateway:{" "}
                  <code className="bg-base-300 px-1 rounded text-xs">
                    {projectConfig.gateway_url}
                  </code>
                </p>
                <ProjectServicesCard config={projectConfig} />
              </div>
            </div>
          )}
        </div>
      </main>
    </div>
  );
}
