import { useState, useEffect } from "react";
import { useParams, Link } from "react-router-dom";
import { api, type ProjectConfig } from "../lib/api";
import { FunctionCard } from "../components/functions/FunctionCard";
import { ProjectServicesCard } from "../components/projects/ProjectServicesCard";

interface Project {
  id: string;
  name: string;
  slug: string;
  owner_id: string;
  created_at: string;
  updated_at: string;
}

interface Function {
  id: string;
  project_id: string;
  function_service_id: string;
  name: string;
  runtime: string;
  created_at: string;
  updated_at: string;
}

export function ProjectDetail() {
  const { id } = useParams<{ id: string }>();
  const [project, setProject] = useState<Project | null>(null);
  const [functions, setFunctions] = useState<Function[]>([]);
  const [projectConfig, setProjectConfig] = useState<ProjectConfig | null>(
    null
  );
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    if (id) {
      loadProject();
      loadFunctions();
      loadProjectConfig();
    }
  }, [id]);

  const loadProject = async () => {
    try {
      const data = await api.getProject(id!);
      setProject(data as Project);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load project");
    }
  };

  const loadFunctions = async () => {
    try {
      const data = await api.listFunctions(id!);
      setFunctions(data as Function[]);
    } catch (err) {
      console.error("Failed to load functions:", err);
    } finally {
      setLoading(false);
    }
  };

  const loadProjectConfig = async () => {
    try {
      const data = await api.getProjectConfig(id!);
      setProjectConfig(data);
    } catch (err) {
      console.error("Failed to load project config:", err);
    }
  };

  if (loading && !project) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="spinner"></div>
      </div>
    );
  }

  if (error || !project) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="card max-w-md">
          <div className="card-body text-center">
            <p className="text-error-600 mb-4">
              {error || "Project not found"}
            </p>
            <Link to="/dashboard" className="link">
              Back to Dashboard
            </Link>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white border-b border-gray-200">
        <div className="container-custom py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <Link
                to="/dashboard"
                className="text-gray-600 hover:text-gray-900"
              >
                ← Back
              </Link>
              <h1 className="text-xl font-bold">{project.name}</h1>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="container-custom py-8">
        <div className="mb-6">
          <div className="card">
            <div className="card-body">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm text-gray-600 mb-1">Slug</p>
                  <code className="text-sm bg-gray-100 px-2 py-1 rounded">
                    {project.slug}
                  </code>
                </div>
                <div>
                  <p className="text-sm text-gray-600 mb-1">Created</p>
                  <p className="text-sm">
                    {new Date(project.created_at).toLocaleDateString()}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>

        {projectConfig && (
          <>
            <h2 className="text-2xl font-bold mb-4">Services</h2>
            <p className="text-gray-600 mb-6">
              Gateway:{" "}
              <code className="text-sm bg-gray-100 px-1.5 py-0.5 rounded">
                {projectConfig.gateway_url}
              </code>
            </p>
            <ProjectServicesCard config={projectConfig} />
            <div className="my-8" />
          </>
        )}

        <div className="flex items-center justify-between mb-6">
          <h2 className="text-2xl font-bold">Functions</h2>
        </div>

        {loading ? (
          <div className="flex justify-center py-12">
            <div className="spinner"></div>
          </div>
        ) : functions?.length === 0 ? (
          <div className="card">
            <div className="card-body text-center py-12">
              <p className="text-gray-600 mb-4">No functions deployed yet</p>
              <div className="bg-gray-50 rounded-lg p-6 text-left max-w-2xl mx-auto">
                <h3 className="font-semibold mb-2">Deploy via CLI</h3>
                <p className="text-sm text-gray-600 mb-4">
                  Use the BunBase CLI to deploy functions to this project:
                </p>
                <pre className="bg-gray-900 text-gray-100 p-4 rounded-lg text-sm overflow-x-auto">
                  <code>
                    {`# Login to platform
bunbase login

# Set active project
bunbase projects use ${project.id}

# Deploy a function
bunbase functions deploy hello-world`}
                  </code>
                </pre>
              </div>
            </div>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {functions?.map((fn) => (
              <FunctionCard key={fn.id} function={fn} projectId={project.id} />
            ))}
          </div>
        )}
      </main>
    </div>
  );
}
