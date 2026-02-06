import { useState, useEffect, useMemo } from "react";
import { useParams, Link } from "react-router-dom";
import {
  Globe,
  Database,
  Clock,
  Pencil,
  FileText,
  MoreVertical,
  Zap,
} from "lucide-react";
import { api } from "../lib/api";
import {
  DUMMY_METRICS,
  type FunctionRow,
  type ApiFunction,
  formatLastDeployed,
} from "../lib/dummyFunctions";

const PAGE_SIZE = 4;
const TIME_RANGES = [
  { value: "1h", label: "Last 1 Hour" },
  { value: "24h", label: "Last 24 Hours" },
  { value: "7d", label: "Last 7 Days" },
] as const;

export function Functions() {
  const { id } = useParams<{ id: string }>();
  const [functions, setFunctions] = useState<ApiFunction[]>([]);
  const [loading, setLoading] = useState(true);
  const [timeRange, setTimeRange] = useState<string>("24h");
  const [currentPage, setCurrentPage] = useState(0);

  useEffect(() => {
    if (id) {
      loadFunctions();
    }
  }, [id]);

  const loadFunctions = async () => {
    try {
      const data = await api.listFunctions(id as string);
      setFunctions(Array.isArray(data) ? (data as ApiFunction[]) : []);
    } catch (err) {
      console.error("Failed to load functions:", err);
      setFunctions([]);
    } finally {
      setLoading(false);
    }
  };

  // Transform API functions to FunctionRow format
  const rows = useMemo(() => {
    if (!functions || functions.length === 0) {
      return [];
    }
    return functions.map((fn) => ({
      id: fn.id,
      name: fn.name,
      runtime: fn.runtime,
      trigger: fn.trigger,
      pathOrCron: fn.path_or_cron,
      status: fn.status,
      lastDeployed: formatLastDeployed(fn.updated_at),
      project_id: fn.project_id,
      function_service_id: fn.function_service_id,
      created_at: fn.created_at,
      updated_at: fn.updated_at,
    }));
  }, [functions]);

  const totalCount = rows.length;
  const totalPages = Math.max(1, Math.ceil(totalCount / PAGE_SIZE));
  const start = currentPage * PAGE_SIZE;
  const pageRows = rows.slice(start, start + PAGE_SIZE);
  const activeCount = rows.filter((r) => r.status === "active").length;
  const deployingCount = rows.filter((r) => r.status === "deploying").length;

  const metrics = useMemo(() => {
    const total = totalCount || DUMMY_METRICS.totalFunctions;
    return {
      totalFunctions: total,
      invocations24h: DUMMY_METRICS.invocations24h,
      invocationsTrend: DUMMY_METRICS.invocationsTrend,
      invocationsSubtitle: DUMMY_METRICS.invocationsSubtitle,
      errorRate: DUMMY_METRICS.errorRate,
      errorRateTrend: DUMMY_METRICS.errorRateTrend,
      errorCountSubtitle: DUMMY_METRICS.errorCountSubtitle,
    };
  }, [totalCount]);

  const TriggerBadge = ({ row }: { row: FunctionRow }) => {
    if (row.trigger === "http") {
      return (
        <span className="badge badge-ghost badge-sm gap-1">
          <Globe className="w-3 h-3" />
          HTTP
        </span>
      );
    }
    if (row.trigger === "database") {
      return (
        <span className="badge badge-ghost badge-sm gap-1">
          <Database className="w-3 h-3" />
          Database
        </span>
      );
    }
    return (
      <span className="badge badge-ghost badge-sm gap-1">
        <Clock className="w-3 h-3" />
        Scheduled
      </span>
    );
  };

  const StatusCell = ({ status }: { status: FunctionRow["status"] }) => {
    const dotClass =
      status === "active"
        ? "bg-success"
        : status === "deploying"
          ? "bg-warning"
          : "bg-error";
    const label =
      status === "active"
        ? "Active"
        : status === "deploying"
          ? "Deploying"
          : "Failed";
    return (
      <span className="flex items-center gap-2">
        <span className={`w-2 h-2 rounded-full ${dotClass}`} aria-hidden />
        {label}
      </span>
    );
  };

  return (
    <div className="flex-1 min-h-0 flex flex-col">
      {/* Header */}
      <div className="flex flex-wrap items-start justify-between gap-4 mb-6">
        <div>
          <h1 className="text-2xl font-bold text-base-content">Functions</h1>
          <p className="text-sm text-base-content/70 mt-0.5">
            Manage and monitor your serverless logic.
          </p>
        </div>
        <select
          className="select select-bordered select-sm"
          value={timeRange}
          onChange={(e) => setTimeRange(e.target.value)}
          aria-label="Time range"
        >
          {TIME_RANGES.map(({ value, label }) => (
            <option key={value} value={value}>
              {label}
            </option>
          ))}
        </select>
      </div>

      {/* Metric cards */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-8">
        <div className="card bg-base-100 shadow-md">
          <div className="card-body p-4">
            <div className="flex items-center justify-between">
              <span className="text-2xl font-bold">{metrics.totalFunctions}</span>
              <span className="text-xs text-success font-medium">+2</span>
            </div>
            <p className="text-xs text-base-content/60">Total Functions</p>
            <p className="text-xs text-base-content/50">
              All runtimes active.
            </p>
          </div>
        </div>
        <div className="card bg-base-100 shadow-md">
          <div className="card-body p-4">
            <div className="flex items-center justify-between">
              <span className="text-2xl font-bold">
                {metrics.invocations24h}
              </span>
              <span className="text-xs text-success font-medium">
                {metrics.invocationsTrend}
              </span>
            </div>
            <p className="text-xs text-base-content/60">Invocations (24h)</p>
            <p className="text-xs text-base-content/50">
              {metrics.invocationsSubtitle}
            </p>
          </div>
        </div>
        <div className="card bg-base-100 shadow-md">
          <div className="card-body p-4">
            <div className="flex items-center justify-between">
              <span className="text-2xl font-bold">{metrics.errorRate}</span>
              <span className="text-xs text-error font-medium">
                {metrics.errorRateTrend}
              </span>
            </div>
            <p className="text-xs text-base-content/60">Error Rate</p>
            <p className="text-xs text-base-content/50">
              {metrics.errorCountSubtitle}
            </p>
          </div>
        </div>
      </div>

      {/* Deployed Functions */}
      <div className="flex-1 min-h-0 flex flex-col">
        <h2 className="text-lg font-semibold mb-1">Deployed Functions</h2>
        <p className="text-sm text-base-content/60 mb-4">
          • {activeCount} Active • {deployingCount} Deploying
        </p>

        {loading ? (
          <div className="flex justify-center py-12">
            <span className="loading loading-spinner loading-lg" />
          </div>
        ) : rows.length === 0 ? (
          <div className="card bg-base-100 shadow-md">
            <div className="card-body text-center py-12">
              <p className="text-base-content/70 mb-4">
                No functions deployed yet
              </p>
              <div className="bg-base-200 rounded-lg p-6 text-left max-w-2xl mx-auto">
                <h3 className="font-semibold mb-2">Deploy via CLI</h3>
                <pre className="bg-base-content text-base-100 p-4 rounded-lg text-sm overflow-x-auto">
                  bunbase projects use {id}
                  {"\n"}bunbase functions deploy {"<name>"}
                </pre>
              </div>
            </div>
          </div>
        ) : (
          <>
            <div className="card bg-base-100 shadow-md flex-1 min-h-0 overflow-hidden">
              <div className="overflow-x-auto overflow-y-auto min-h-0">
                <table className="table table-pin-rows">
                  <thead>
                    <tr>
                      <th className="font-semibold">FUNCTION NAME</th>
                      <th className="font-semibold">TRIGGER</th>
                      <th className="font-semibold">RUNTIME</th>
                      <th className="font-semibold">LAST DEPLOYED</th>
                      <th className="font-semibold">STATUS</th>
                      <th className="font-semibold">ACTIONS</th>
                    </tr>
                  </thead>
                  <tbody>
                    {pageRows.map((row) => (
                      <tr key={row.id}>
                        <td>
                          <div>
                            <p className="font-medium">{row.name}</p>
                            <p className="text-xs text-base-content/60">
                              {row.pathOrCron}
                            </p>
                          </div>
                        </td>
                        <td>
                          <TriggerBadge row={row} />
                        </td>
                        <td>
                          <span className="badge badge-outline badge-sm">
                            {row.runtime}
                          </span>
                        </td>
                        <td className="text-base-content/80">
                          {row.lastDeployed}
                        </td>
                        <td>
                          <StatusCell status={row.status} />
                        </td>
                        <td>
                          <div className="flex items-center gap-2">
                            <button
                              type="button"
                              className="btn btn-ghost btn-square btn-sm"
                              aria-label="Edit"
                              title="Edit"
                            >
                              <Pencil className="w-4 h-4" />
                            </button>
                            <Link
                              to={`/projects/${id}/functions/logs?function=${encodeURIComponent(row.name)}`}
                              className="btn btn-ghost btn-square btn-sm"
                              aria-label="Logs"
                              title="Logs"
                            >
                              <FileText className="w-4 h-4" />
                            </Link>
                            <div className="dropdown dropdown-end">
                              <button
                                type="button"
                                tabIndex={0}
                                className="btn btn-ghost btn-square btn-sm"
                                aria-label="More options"
                              >
                                <MoreVertical className="w-4 h-4" />
                              </button>
                              <ul
                                tabIndex={0}
                                className="dropdown-content menu bg-base-200 rounded-box z-10 w-48 p-2 shadow"
                              >
                                <li>
                                  <button type="button">Edit</button>
                                </li>
                                <li>
                                  <Link
                                    to={`/projects/${id}/functions/logs?function=${encodeURIComponent(row.name)}`}
                                  >
                                    View logs
                                  </Link>
                                </li>
                              </ul>
                            </div>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>

            {/* Pagination */}
            <div className="flex flex-wrap items-center justify-between gap-4 mt-4">
              <p className="text-sm text-base-content/70">
                Showing {Math.min(PAGE_SIZE, pageRows.length)} of {totalCount}{" "}
                functions.
              </p>
              <div className="join">
                <button
                  type="button"
                  className="btn btn-sm join-item"
                  disabled={currentPage === 0}
                  onClick={() => setCurrentPage((p) => Math.max(0, p - 1))}
                >
                  Previous
                </button>
                <button
                  type="button"
                  className="btn btn-sm join-item"
                  disabled={currentPage >= totalPages - 1}
                  onClick={() =>
                    setCurrentPage((p) => Math.min(totalPages - 1, p + 1))
                  }
                >
                  Next
                </button>
              </div>
            </div>
          </>
        )}

        {/* Pro tip */}
        <div className="mt-8 flex items-start gap-3 p-4 rounded-lg bg-base-100 border border-base-300">
          <Zap className="w-5 h-5 text-primary shrink-0 mt-0.5" />
          <div>
            <p className="font-medium text-base-content">Pro Tip</p>
            <p className="text-sm text-base-content/70 mt-0.5">
              You can deploy local functions directly using the{" "}
              <code className="bg-base-200 px-1.5 py-0.5 rounded text-sm">
                bunbase deploy
              </code>{" "}
              command from your terminal.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
