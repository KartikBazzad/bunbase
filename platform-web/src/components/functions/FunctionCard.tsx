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
}

export function FunctionCard({ function: fn, projectId }: FunctionCardProps) {
  return (
    <div className="card">
      <div className="card-body">
        <div className="flex items-start justify-between mb-2">
          <h3 className="text-lg font-semibold">{fn.name}</h3>
          <span className={`badge-${fn.runtime === 'quickjs-ng' ? 'success' : 'primary'}`}>
            {fn.runtime}
          </span>
        </div>
        <p className="text-sm text-gray-600 mb-2">
          Service ID: <code className="text-xs bg-gray-100 px-1.5 py-0.5 rounded">{fn.function_service_id}</code>
        </p>
        <p className="text-xs text-gray-500">
          Deployed {new Date(fn.created_at).toLocaleDateString()}
        </p>
      </div>
    </div>
  );
}
