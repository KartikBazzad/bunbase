import type { ProjectConfig } from "../../lib/api";

interface ProjectServicesCardProps {
  config: ProjectConfig;
}

function CopyableValue({ label, value }: { label: string; value: string }) {
  const copy = () => {
    navigator.clipboard.writeText(value);
  };
  return (
    <div className="mb-2">
      <p className="text-xs text-base-content/50 mb-0.5">{label}</p>
      <div className="flex items-center gap-2">
        <code className="text-sm bg-base-300 px-2 py-1 rounded flex-1 truncate">
          {value}
        </code>
        <button
          type="button"
          onClick={copy}
          className="text-xs text-base-content/70 hover:text-base-content whitespace-nowrap"
        >
          Copy
        </button>
      </div>
    </div>
  );
}

export function ProjectServicesCard({ config }: ProjectServicesCardProps) {
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
      <div className="card bg-base-100 shadow-md">
        <div className="card-body">
          <h3 className="font-semibold mb-2">KV (Bunder)</h3>
          <p className="text-sm text-base-content/70 mb-2">
            Project-scoped key-value store. Use to path below for HTTP API.
          </p>
          <CopyableValue label="Path" value={config.kv.path} />
        </div>
      </div>
      <div className="card bg-base-100 shadow-md">
        <div className="card-body">
          <h3 className="font-semibold mb-2">Bundoc</h3>
          <p className="text-sm text-base-content/70 mb-2">
            Document database. Base path for collections.
          </p>
          <CopyableValue
            label="Documents path"
            value={config.bundoc.documents_path}
          />
        </div>
      </div>
      <div className="card bg-base-100 shadow-md">
        <div className="card-body">
          <h3 className="font-semibold mb-2">Buncast</h3>
          <p className="text-sm text-base-content/70 mb-2">
            Pub/sub. Subscribe with topic prefix below.
          </p>
          <CopyableValue
            label="Topic prefix"
            value={config.buncast.topic_prefix}
          />
          <p className="text-xs text-base-content/50 mt-1">
            e.g. {config.buncast.topic_prefix}events
          </p>
        </div>
      </div>
      <div className="card bg-base-100 shadow-md">
        <div className="card-body">
          <h3 className="font-semibold mb-2">Functions</h3>
          <p className="text-sm text-base-content/70 mb-2">
            Invoke endpoint for serverless functions.
          </p>
          <CopyableValue
            label="Invoke path"
            value={config.functions.invoke_path}
          />
        </div>
      </div>
    </div>
  );
}
