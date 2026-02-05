import { useState, useEffect, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { api } from "../lib/api";

interface Project {
  id: string;
  name: string;
  slug: string;
  owner_id: string;
  public_api_key?: string | null;
  created_at: string;
  updated_at: string;
}

export function Settings() {
  const { id: projectId } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const [project, setProject] = useState<Project | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const [name, setName] = useState("");
  const [nameSaving, setNameSaving] = useState(false);
  const [nameError, setNameError] = useState("");
  const [nameSuccess, setNameSuccess] = useState(false);

  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [deleteConfirmName, setDeleteConfirmName] = useState("");
  const [deleteLoading, setDeleteLoading] = useState(false);
  const [deleteError, setDeleteError] = useState("");

  const [regenerateLoading, setRegenerateLoading] = useState(false);
  const [regenerateError, setRegenerateError] = useState("");
  const [showRegenerateModal, setShowRegenerateModal] = useState(false);
  const [newKeyAfterRegenerate, setNewKeyAfterRegenerate] = useState<string | null>(null);
  const [copySuccess, setCopySuccess] = useState(false);

  const loadProject = useCallback(async () => {
    if (!projectId) return;
    setLoading(true);
    setError("");
    try {
      const data = (await api.getProject(projectId)) as Project;
      setProject(data);
      setName(data.name);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load project");
    } finally {
      setLoading(false);
    }
  }, [projectId]);

  useEffect(() => {
    if (projectId) loadProject();
  }, [projectId, loadProject]);

  const handleSaveName = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!projectId || !name.trim()) {
      setNameError("Project name is required.");
      return;
    }
    setNameError("");
    setNameSuccess(false);
    setNameSaving(true);
    try {
      const updated = (await api.updateProject(
        projectId,
        name.trim(),
      )) as Project;
      setProject(updated);
      setName(updated.name);
      setNameSuccess(true);
      setTimeout(() => setNameSuccess(false), 3000);
    } catch (err) {
      setNameError(
        err instanceof Error ? err.message : "Failed to update project",
      );
    } finally {
      setNameSaving(false);
    }
  };

  const handleCopyKey = async (key: string) => {
    try {
      await navigator.clipboard.writeText(key);
      setCopySuccess(true);
      setTimeout(() => setCopySuccess(false), 2000);
    } catch {
      setRegenerateError("Failed to copy");
    }
  };

  const handleRegenerateClick = () => {
    setRegenerateError("");
    setNewKeyAfterRegenerate(null);
    setShowRegenerateModal(true);
  };

  const handleRegenerateConfirm = async () => {
    if (!projectId) return;
    setRegenerateLoading(true);
    setRegenerateError("");
    try {
      const res = await api.regenerateProjectApiKey(projectId);
      setProject((prev) => (prev ? { ...prev, public_api_key: res.api_key } : null));
      setNewKeyAfterRegenerate(res.api_key);
      setShowRegenerateModal(false);
      await loadProject();
    } catch (err) {
      setRegenerateError(err instanceof Error ? err.message : "Failed to regenerate key");
    } finally {
      setRegenerateLoading(false);
    }
  };

  const handleDeleteClick = () => {
    setDeleteConfirmName("");
    setDeleteError("");
    setShowDeleteModal(true);
  };

  const handleDeleteConfirm = async () => {
    if (!projectId || !project) return;
    if (deleteConfirmName.trim() !== project.name) {
      setDeleteError("Type the project name exactly to confirm.");
      return;
    }
    setDeleteError("");
    setDeleteLoading(true);
    try {
      await api.deleteProject(projectId);
      setShowDeleteModal(false);
      navigate("/dashboard");
    } catch (err) {
      setDeleteError(
        err instanceof Error ? err.message : "Failed to delete project",
      );
    } finally {
      setDeleteLoading(false);
    }
  };

  if (!projectId) {
    return (
      <div className="min-h-screen bg-base-200 flex flex-col">
        <main className="container mx-auto px-4 sm:px-6 lg:px-8 max-w-7xl py-8">
          <p className="text-base-content/70">No project selected.</p>
        </main>
      </div>
    );
  }

  if (loading && !project) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <span className="loading loading-spinner loading-lg" />
      </div>
    );
  }

  if (error || !project) {
    return (
      <div className="min-h-screen bg-base-200 flex flex-col">
        <main className="container mx-auto px-4 sm:px-6 lg:px-8 max-w-7xl py-8">
          <div className="alert alert-error">
            {error || "Project not found."}
          </div>
        </main>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-base-200 flex flex-col">
      <main className="container mx-auto px-4 sm:px-6 lg:px-8 max-w-7xl py-8">
        <div className="mb-6">
          <h1 className="text-2xl font-bold">Settings</h1>
          <p className="text-base-content/70 mt-1">
            Manage project name and delete project.
          </p>
        </div>

        <div className="space-y-6">
          {/* General */}
          <div className="card bg-base-100 shadow-md">
            <div className="card-body">
              <h2 className="card-title text-lg">General</h2>
              <form className="space-y-4" onSubmit={handleSaveName}>
                <div className="form-control">
                  <label className="label" htmlFor="project-name">
                    <span className="label-text">Project name</span>
                  </label>
                  <input
                    id="project-name"
                    type="text"
                    className="input input-bordered w-full max-w-md"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    disabled={nameSaving}
                    placeholder="My Project"
                  />
                </div>
                <div className="flex flex-wrap items-center gap-3">
                  <button
                    type="submit"
                    className="btn btn-primary"
                    disabled={nameSaving || name.trim() === project.name}
                  >
                    {nameSaving ? (
                      <>
                        <span className="loading loading-spinner loading-sm" />
                        Saving…
                      </>
                    ) : (
                      "Save"
                    )}
                  </button>
                  {nameSuccess && (
                    <span className="text-sm text-success">Saved.</span>
                  )}
                </div>
                {nameError && (
                  <div className="alert alert-error text-sm">{nameError}</div>
                )}
              </form>
              <div className="mt-4 pt-4 border-t border-base-300 space-y-2">
                <p className="text-sm text-base-content/60">Project ID</p>
                <code className="text-xs bg-base-200 px-2 py-1 rounded block break-all">
                  {project.id}
                </code>
                <p className="text-sm text-base-content/60 mt-2">Slug</p>
                <code className="text-xs bg-base-200 px-2 py-1 rounded">
                  {project.slug}
                </code>
              </div>
            </div>
          </div>

          {/* API key */}
          <div className="card bg-base-100 shadow-md">
            <div className="card-body">
              <h2 className="card-title text-lg">API key</h2>
              <p className="text-base-content/70 text-sm">
                Use this key in the BunBase SDK or in request headers (
                <code className="bg-base-200 px-1 rounded">X-Bunbase-Client-Key</code>
                ) to access this project&apos;s database and functions.
              </p>
              {project.public_api_key ? (
                <div className="flex flex-wrap items-center gap-2 mt-2">
                  <code className="text-xs bg-base-200 px-2 py-1 rounded break-all flex-1 min-w-0">
                    {project.public_api_key}
                  </code>
                  <button
                    type="button"
                    className="btn btn-sm btn-ghost"
                    onClick={() => handleCopyKey(project.public_api_key!)}
                  >
                    {copySuccess ? "Copied" : "Copy"}
                  </button>
                  <button
                    type="button"
                    className="btn btn-sm btn-outline btn-warning"
                    onClick={handleRegenerateClick}
                    disabled={regenerateLoading}
                  >
                    {regenerateLoading ? "Regenerating…" : "Regenerate"}
                  </button>
                </div>
              ) : (
                <div className="mt-2">
                  <p className="text-sm text-base-content/60">No API key set.</p>
                  <button
                    type="button"
                    className="btn btn-sm btn-primary mt-2"
                    onClick={handleRegenerateClick}
                    disabled={regenerateLoading}
                  >
                    {regenerateLoading ? "Generating…" : "Generate API key"}
                  </button>
                </div>
              )}
              {regenerateError && (
                <div className="alert alert-error text-sm mt-2">{regenerateError}</div>
              )}
            </div>
          </div>

          {/* Danger zone */}
          <div className="card bg-base-100 shadow-md border border-error/20">
            <div className="card-body">
              <h2 className="card-title text-lg text-error">Danger zone</h2>
              <p className="text-base-content/70 text-sm">
                Deleting this project is permanent. All data and configuration
                will be removed.
              </p>
              <div className="card-actions justify-start mt-2">
                <button
                  type="button"
                  className="btn btn-error btn-outline"
                  onClick={handleDeleteClick}
                >
                  Delete project
                </button>
              </div>
            </div>
          </div>
        </div>
      </main>

      {/* Regenerate API key confirmation */}
      <dialog className={`modal ${showRegenerateModal ? "modal-open" : ""}`}>
        <div className="modal-box">
          <h3 className="font-bold text-lg">Regenerate API key</h3>
          <p className="py-2 text-base-content/70">
            The current API key will stop working. Any clients using it will need to be updated with the new key. Continue?
          </p>
          <div className="modal-action">
            <button
              type="button"
              className="btn btn-ghost"
              onClick={() => {
                setShowRegenerateModal(false);
                setRegenerateError("");
              }}
              disabled={regenerateLoading}
            >
              Cancel
            </button>
            <button
              type="button"
              className="btn btn-warning"
              onClick={handleRegenerateConfirm}
              disabled={regenerateLoading}
            >
              {regenerateLoading ? "Regenerating…" : "Regenerate"}
            </button>
          </div>
        </div>
        <form method="dialog" className="modal-backdrop">
          <button
            type="button"
            onClick={() => {
              setShowRegenerateModal(false);
              setRegenerateError("");
            }}
          >
            close
          </button>
        </form>
      </dialog>

      {/* New key shown once after regenerate */}
      <dialog className={`modal ${newKeyAfterRegenerate ? "modal-open" : ""}`}>
        <div className="modal-box">
          <h3 className="font-bold text-lg">New API key</h3>
          <p className="py-2 text-base-content/70 text-sm">
            Copy this key now. It won&apos;t be shown again.
          </p>
          <div className="flex flex-wrap items-center gap-2">
            <code className="text-xs bg-base-200 px-2 py-1 rounded break-all flex-1 min-w-0">
              {newKeyAfterRegenerate}
            </code>
            <button
              type="button"
              className="btn btn-sm"
              onClick={() => newKeyAfterRegenerate && handleCopyKey(newKeyAfterRegenerate)}
            >
              Copy
            </button>
          </div>
          <div className="modal-action">
            <button
              type="button"
              className="btn btn-primary"
              onClick={() => setNewKeyAfterRegenerate(null)}
            >
              Done
            </button>
          </div>
        </div>
        <form method="dialog" className="modal-backdrop">
          <button type="button" onClick={() => setNewKeyAfterRegenerate(null)}>
            close
          </button>
        </form>
      </dialog>

      {/* Delete confirmation modal */}
      <dialog className={`modal ${showDeleteModal ? "modal-open" : ""}`}>
        <div className="modal-box">
          <h3 className="font-bold text-lg">Delete project</h3>
          <p className="py-2 text-base-content/70">
            This action cannot be undone. Type the project name below to
            confirm.
          </p>
          <div className="form-control mt-4">
            <input
              type="text"
              className="input input-bordered w-full"
              placeholder={project.name}
              value={deleteConfirmName}
              onChange={(e) => setDeleteConfirmName(e.target.value)}
              disabled={deleteLoading}
            />
          </div>
          {deleteError && (
            <div className="alert alert-error text-sm mt-3">{deleteError}</div>
          )}
          <div className="modal-action">
            <button
              type="button"
              className="btn btn-ghost"
              onClick={() => {
                setShowDeleteModal(false);
                setDeleteError("");
                setDeleteConfirmName("");
              }}
              disabled={deleteLoading}
            >
              Cancel
            </button>
            <button
              type="button"
              className="btn btn-error"
              onClick={handleDeleteConfirm}
              disabled={
                deleteLoading || deleteConfirmName.trim() !== project.name
              }
            >
              {deleteLoading ? (
                <>
                  <span className="loading loading-spinner loading-sm" />
                  Deleting…
                </>
              ) : (
                "Delete project"
              )}
            </button>
          </div>
        </div>
        <form method="dialog" className="modal-backdrop">
          <button
            type="button"
            onClick={() => {
              setShowDeleteModal(false);
              setDeleteError("");
              setDeleteConfirmName("");
            }}
          >
            close
          </button>
        </form>
      </dialog>
    </div>
  );
}
