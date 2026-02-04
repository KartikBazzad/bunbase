import { useState } from "react";
import { FunctionInvoker } from "./FunctionInvoker";
import { api } from "../../lib/api";
import { Play, Trash2, Zap } from 'lucide-react';

interface Function {
  id: string;
  project_id: string;
  function_service_id: string;
  name: string;
  runtime: string;
  created_at: string;
  updated_at: string;
}

interface FunctionCardProps {
  function: Function;
  projectId: string;
  onDelete?: () => void;
}

export function FunctionCard({
  function: fn,
  projectId,
  onDelete,
}: FunctionCardProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  const handleDelete = async () => {
    if (!confirm(`Are you sure you want to delete function "${fn.name}"?`))
      return;

    setIsDeleting(true);
    try {
      await api.deleteFunction(projectId, fn.id);
      if (onDelete) onDelete();
    } catch (err) {
      console.error("Failed to delete function:", err);
      alert("Failed to delete function");
      setIsDeleting(false);
    }
  };

  return (
    <div className="card bg-base-100 shadow-md transition-all">
      <div className="card-body">
        <div className="flex items-start justify-between mb-2">
          <div className="flex items-center gap-2">
            <Zap className="w-5 h-5 text-primary" />
            <h3 className="text-lg font-semibold">{fn.name}</h3>
          </div>
          <div
            className={`badge ${fn.runtime === "quickjs" ? "badge-success" : "badge-primary"}`}
          >
            {fn.runtime}
          </div>
        </div>
        <p className="text-sm text-base-content/70 mb-2">
          Service ID:{" "}
          <code className="text-xs bg-base-300 px-1.5 py-0.5 rounded">
            {fn.function_service_id}
          </code>
        </p>
        <p className="text-xs text-base-content/50 mb-4">
          Deployed {new Date(fn.created_at).toLocaleDateString()}
        </p>

        <div className="flex gap-2 border-t pt-3 mt-2">
          <button
            className="btn btn-secondary btn-sm flex-1"
            onClick={() => setIsExpanded(!isExpanded)}
          >
            <Play className="w-3 h-3 mr-1" />
            {isExpanded ? "Close" : "Test"}
          </button>
          <button
            className="btn btn-error btn-sm"
            onClick={handleDelete}
            disabled={isDeleting}
          >
            {isDeleting ? (
              <span className="loading loading-spinner loading-sm"></span>
            ) : (
              <>
                <Trash2 className="w-3 h-3" />
                Delete
              </>
            )}
          </button>
        </div>

        {isExpanded && (
          <FunctionInvoker projectId={projectId} functionName={fn.name} />
        )}
      </div>
    </div>
  );
}
