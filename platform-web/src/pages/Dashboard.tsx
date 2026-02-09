import { useState, useEffect } from "react";
import { api } from "../lib/api";
import { Link } from "react-router-dom";
import { Plus, FolderKanban } from "lucide-react";
import { useAuth } from "../hooks/useAuth";
import { useInstanceStatus } from "../hooks/useInstanceStatus";

interface Project {
  id: string;
  name: string;
  slug: string;
  owner_id: string;
  created_at: string;
  updated_at: string;
}

export function Dashboard() {
  const { user } = useAuth();
  const { status } = useInstanceStatus();
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);

  const canCreateProject =
    !status ||
    status.deployment_mode !== "self_hosted" ||
    user?.is_instance_admin === true;

  useEffect(() => {
    loadProjects();
  }, []);

  const loadProjects = async () => {
    try {
      const data = await api.listProjects();
      setProjects(data as Project[]);
    } catch (error) {
      console.error("Failed to load projects:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateProject = async (name: string) => {
    try {
      await api.createProject(name);
      await loadProjects();
      setShowCreateModal(false);
    } catch (error) {
      console.error("Failed to create project:", error);
      throw error;
    }
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold">Projects</h2>
        {canCreateProject ? (
          <button
            className="btn btn-primary"
            onClick={() => setShowCreateModal(true)}
          >
            <Plus className="w-4 h-4 mr-2" />
            Create Project
          </button>
        ) : (
          <p className="text-sm text-base-content/70">
            Only instance administrators can create projects.
          </p>
        )}
      </div>

      {loading ? (
        <div className="flex justify-center py-12">
          <span className="loading loading-spinner loading-lg"></span>
        </div>
      ) : projects?.length === 0 ? (
        <div className="card max-w-md mx-auto bg-base-100 shadow-xl">
          <div className="card-body text-center py-12">
            <div className="mx-auto mb-4 p-4 bg-base-200 rounded-full w-16 h-16 flex items-center justify-center">
              <FolderKanban className="w-8 h-8 text-base-content/50" />
            </div>
            <p className="text-base-content/70 mb-4">No projects yet</p>
            {canCreateProject ? (
              <button
                className="btn btn-primary"
                onClick={() => setShowCreateModal(true)}
              >
                <Plus className="w-4 h-4 mr-2" />
                Create Your First Project
              </button>
            ) : (
              <p className="text-sm text-base-content/70">
                Only instance administrators can create projects.
              </p>
            )}
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {projects?.map((project) => (
            <Link
              key={project.id}
              to={`/projects/${project.id}/overview`}
              className="card bg-base-100 shadow-md hover:shadow-lg transition-shadow cursor-pointer h-full group"
            >
              <div className="card-body">
                <div className="flex items-start justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <FolderKanban className="w-5 h-5 text-primary group-hover:scale-110 transition-transform" />
                    <h3 className="text-lg font-semibold">{project.name}</h3>
                  </div>
                  <div className="badge badge-primary">Active</div>
                </div>
                <p className="text-sm text-base-content/70 mb-4">
                  Slug:{" "}
                  <code className="text-xs bg-base-300 px-1.5 py-0.5 rounded">
                    {project.slug}
                  </code>
                </p>
                <div className="flex items-center text-xs text-base-content/50">
                  Created {new Date(project.created_at).toLocaleDateString()}
                </div>
              </div>
            </Link>
          ))}
        </div>
      )}

      {/* Create Project Modal */}
      <dialog className={`modal ${showCreateModal ? "modal-open" : ""}`}>
        <div className="modal-box">
          <div className="flex items-center gap-2 mb-4">
            <FolderKanban className="w-6 h-6 text-primary" />
            <h3 className="font-bold text-lg">Create New Project</h3>
          </div>

          <form
            onSubmit={async (e) => {
              e.preventDefault();
              const nameInput = e.currentTarget.elements.namedItem(
                "project-name"
              ) as HTMLInputElement;
              if (!nameInput.value.trim()) return;
              await handleCreateProject(nameInput.value.trim());
            }}
            className="space-y-4"
          >
            <div className="form-control">
              <label className="label" htmlFor="project-name">
                <span className="label-text">Project Name</span>
                <span className="label-text-alt text-error">*</span>
              </label>
              <input
                id="project-name"
                type="text"
                placeholder="my-awesome-project"
                className="input input-bordered w-full"
                autoFocus
                required
              />
              <p className="text-xs text-base-content/50 mt-1">
                A URL-friendly slug will be generated automatically
              </p>
            </div>

            <div className="modal-action">
              <button
                type="button"
                onClick={() => setShowCreateModal(false)}
                className="btn btn-secondary"
              >
                Cancel
              </button>
              <button type="submit" className="btn btn-primary">
                Create
              </button>
            </div>
          </form>
        </div>
        <form method="dialog" className="modal-backdrop">
          <button onClick={() => setShowCreateModal(false)}>close</button>
        </form>
      </dialog>
    </div>
  );
}
