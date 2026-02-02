import { useState, useEffect } from "react";
import { useParams, Link } from "react-router-dom";
import { api, type ProjectConfig } from "../lib/api";
import { FunctionCard } from "../components/functions/FunctionCard";
import { ProjectServicesCard } from "../components/projects/ProjectServicesCard";
import { CollectionList } from "../components/database/CollectionList";
import { DocumentBrowser } from "../components/database/DocumentBrowser";

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
    null,
  );
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [activeTab, setActiveTab] = useState<
    "overview" | "database" | "functions" | "settings"
  >("overview");

  // Database State
  const [selectedCollection, setSelectedCollection] = useState("");

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

  const TabButton = ({
    id,
    label,
  }: {
    id: typeof activeTab;
    label: string;
  }) => (
    <button
      onClick={() => setActiveTab(id)}
      className={`px-4 py-2 text-sm font-medium border-b-2 mr-4 ${
        activeTab === id
          ? "border-primary-600 text-primary-600"
          : "border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300"
      }`}
    >
      {label}
    </button>
  );

  return (
    <div className="min-h-screen bg-gray-50 flex flex-col">
      {/* Header */}
      <header className="bg-white border-b border-gray-200">
        <div className="container-custom py-4">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-4">
              <Link
                to="/dashboard"
                className="text-gray-600 hover:text-gray-900"
              >
                ← Back
              </Link>
              <h1 className="text-xl font-bold">{project.name}</h1>
              <span className="bg-gray-100 text-xs px-2 py-1 rounded text-gray-500">
                {project.slug}
              </span>
            </div>
          </div>

          <div className="flex">
            <TabButton id="overview" label="Overview" />
            <TabButton id="database" label="Database" />
            <TabButton id="functions" label="Functions" />
            {/* <TabButton id="settings" label="Settings" /> */}
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="container-custom py-8 flex-1">
        {/* OVERVIEW TAB */}
        {activeTab === "overview" && (
          <div className="space-y-6">
            {projectConfig && (
              <>
                <div className="flex justify-between items-center">
                  <h2 className="text-xl font-bold">API Services</h2>
                  <p className="text-sm">
                    Gateway:{" "}
                    <code className="bg-gray-200 px-1 rounded">
                      {projectConfig.gateway_url}
                    </code>
                  </p>
                </div>
                <ProjectServicesCard config={projectConfig} />
              </>
            )}
          </div>
        )}

        {/* DATABASE TAB */}
        {activeTab === "database" && (
          <div className="grid grid-cols-12 gap-6 h-[600px]">
            <div className="col-span-3 h-full">
              <CollectionList
                projectId={project.id}
                onSelectCollection={setSelectedCollection}
              />
            </div>
            <div className="col-span-9 h-full">
              <DocumentBrowser
                projectId={project.id}
                collection={selectedCollection}
              />
            </div>
          </div>
        )}

        {/* FUNCTIONS TAB */}
        {activeTab === "functions" && (
          <div>
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-xl font-bold">Deployed Functions</h2>
            </div>
            {functions.length === 0 ? (
              <div className="card">
                <div className="card-body text-center py-12">
                  <p className="text-gray-600 mb-4">
                    No functions deployed yet
                  </p>
                  <div className="bg-gray-50 rounded-lg p-6 text-left max-w-2xl mx-auto">
                    <h3 className="font-semibold mb-2">Deploy via CLI</h3>
                    <pre className="bg-gray-900 text-gray-100 p-4 rounded-lg text-sm overflow-x-auto">
                      <code>
                        {`bunbase projects use ${project.id}\nbunbase functions deploy <name>`}
                      </code>
                    </pre>
                  </div>
                </div>
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {functions.map((fn) => (
                  <FunctionCard
                    key={fn.id}
                    function={fn}
                    projectId={project.id}
                  />
                ))}
              </div>
            )}
          </div>
        )}
      </main>
    </div>
  );
}
