import { useState, useEffect } from "react";
import { FolderKanban, AlertCircle, X, Save } from "lucide-react";

interface CreateProjectModalProps {
  onClose: () => void;
  onCreate: (name: string) => Promise<void>;
}

export function CreateProjectModal({
  onClose,
  onCreate,
}: CreateProjectModalProps) {
  const [name, setName] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    if (!name.trim()) {
      setError("Project name is required");
      return;
    }

    setLoading(true);
    try {
      await onCreate(name.trim());
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create project");
      setLoading(false);
    }
  };

  return (
    <dialog className="modal modal-open">
      <div className="modal-box">
        <div className="flex items-center gap-2 mb-4">
          <FolderKanban className="w-6 h-6 text-primary" />
          <h3 className="font-bold text-lg">Create New Project</h3>
        </div>

        <form
          onSubmit={handleSubmit}
          className="space-y-4"
        >
          {error && (
            <div className="alert alert-error">
              <AlertCircle className="w-5 h-5 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          <div className="form-control">
            <label className="label" htmlFor="project-name">
              <span className="label-text">Project Name</span>
              <span className="label-text-alt text-error">*</span>
            </label>
            <input
              id="project-name"
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="my-awesome-project"
              className="input input-bordered w-full"
              required
              autoFocus
            />
            <p className="text-xs text-base-content/50 mt-1">
              A URL-friendly slug will be generated automatically
            </p>
          </div>

          <div className="modal-action">
            <button
              type="button"
              onClick={onClose}
              disabled={loading}
              className="btn btn-secondary"
            >
              <X className="w-4 h-4 mr-1" />
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading}
              className={`btn btn-primary ${loading ? 'btn-disabled' : ''}`}
            >
              {loading && <span className="loading loading-spinner"></span>}
              <Save className="w-4 h-4 mr-1" />
              {loading ? 'Creating...' : 'Create'}
            </button>
          </div>
        </form>
      </div>
      <form method="dialog" className="modal-backdrop">
        <button onClick={onClose}>close</button>
      </form>
    </dialog>
  );
}
