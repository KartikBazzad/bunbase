import { useState, useEffect, useMemo, useRef } from "react";
import { useParams, useSearchParams, Link } from "react-router-dom";
import { ChevronRight, Download, Trash2, Pause, Filter } from "lucide-react";
import { api } from "../lib/api";
import {
  DUMMY_LOG_ENTRIES,
  type LogEntry,
} from "../lib/dummyFunctions";

const SEVERITIES = ["All Severities", "INFO", "WARN", "ERROR", "DEBUG"] as const;
const TIME_RANGES = [
  { value: "15m", label: "Last 15 minutes", minutes: 15 },
  { value: "1h", label: "Last 1 hour", minutes: 60 },
  { value: "24h", label: "Last 24 hours", minutes: 24 * 60 },
] as const;

const FUNCTION_NAMES = [
  ...new Set(DUMMY_LOG_ENTRIES.map((e) => e.functionName).filter(Boolean)),
] as string[];

function formatTime(iso: string): string {
  const d = new Date(iso);
  const h = d.getHours().toString().padStart(2, "0");
  const m = d.getMinutes().toString().padStart(2, "0");
  const s = d.getSeconds().toString().padStart(2, "0");
  const ms = d.getMilliseconds().toString().padStart(3, "0");
  return `${h}:${m}:${s}.${ms}`;
}

function severityColor(severity: LogEntry["severity"]): string {
  switch (severity) {
    case "ERROR":
      return "text-error";
    case "WARN":
      return "text-warning";
    case "DEBUG":
      return "text-base-content/50";
    default:
      return "text-info";
  }
}

/** Simple JSON syntax highlight: keys and string values */
function JsonBlock({ payload }: { payload: string }) {
  try {
    const parsed = JSON.parse(payload) as Record<string, unknown>;
    const entries = Object.entries(parsed);
    return (
      <pre className="ml-4 mt-1 text-xs overflow-x-auto text-base-content/90">
        {"{"}
        {entries.map(([k, v], i) => (
          <div key={i} className="ml-2">
            <span className="text-primary">"{k}"</span>
            {": "}
            {typeof v === "string" ? (
              <span className="text-secondary">"{v}"</span>
            ) : (
              <span>{String(v)}</span>
            )}
            {i < entries.length - 1 ? "," : ""}
          </div>
        ))}
        {"}"}
      </pre>
    );
  } catch {
    return (
      <pre className="ml-4 mt-1 text-xs overflow-x-auto text-base-content/70">
        {payload}
      </pre>
    );
  }
}

export function FunctionLogs() {
  const { id } = useParams<{ id: string }>();
  const [searchParams] = useSearchParams();
  const initialFunction = searchParams.get("function") ?? "";

  const [logs, setLogs] = useState<LogEntry[]>(() => [...DUMMY_LOG_ENTRIES]);
  const [projectName, setProjectName] = useState("Project");
  const [severity, setSeverity] = useState<string>(SEVERITIES[0]);
  const [functionFilter, setFunctionFilter] = useState(initialFunction);
  const [timeRange, setTimeRange] = useState("15m");
  const [keyword, setKeyword] = useState("");
  const [live, setLive] = useState(true);
  const [paused, setPaused] = useState(false);
  const [autoScroll, setAutoScroll] = useState(true);
  const panelRef = useRef<HTMLDivElement>(null);
  const streamIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    if (id) {
      api
        .getProject(id)
        .then((p: { name?: string }) => setProjectName(p?.name ?? "Project"))
        .catch(() => setProjectName("Project"));
    }
  }, [id]);

  useEffect(() => {
    if (initialFunction) setFunctionFilter(initialFunction);
  }, [initialFunction]);

  useEffect(() => {
    if (!live || paused) {
      if (streamIntervalRef.current) {
        clearInterval(streamIntervalRef.current);
        streamIntervalRef.current = null;
      }
      return;
    }
    streamIntervalRef.current = setInterval(() => {
      setLogs((prev) => {
        const last = prev[prev.length - 1];
        const ts = new Date().toISOString();
        const newEntry: LogEntry = {
          id: `log-live-${Date.now()}`,
          timestamp: ts,
          severity: "INFO",
          requestId: `REQ-${Math.random().toString(36).slice(2, 8).toUpperCase()}`,
          message: "Waiting for new events...",
          functionName: last?.functionName ?? "user-auth-hook",
        };
        return [...prev, newEntry];
      });
    }, 4000);
    return () => {
      if (streamIntervalRef.current) {
        clearInterval(streamIntervalRef.current);
      }
    };
  }, [live, paused]);

  const timeRangeMinutes = TIME_RANGES.find((r) => r.value === timeRange)?.minutes ?? 15;
  const cutoff = Date.now() - timeRangeMinutes * 60 * 1000;

  const filteredLogs = useMemo(() => {
    return logs.filter((entry) => {
      const t = new Date(entry.timestamp).getTime();
      if (t < cutoff) return false;
      if (severity !== "All Severities" && entry.severity !== severity) return false;
      if (functionFilter && entry.functionName !== functionFilter) return false;
      if (keyword) {
        const k = keyword.toLowerCase();
        const match =
          entry.message.toLowerCase().includes(k) ||
          entry.requestId.toLowerCase().includes(k);
        if (!match) return false;
      }
      return true;
    });
  }, [logs, severity, functionFilter, timeRange, timeRangeMinutes, cutoff, keyword]);

  useEffect(() => {
    if (autoScroll && panelRef.current) {
      panelRef.current.scrollTop = panelRef.current.scrollHeight;
    }
  }, [filteredLogs.length, autoScroll]);

  const exportCsv = () => {
    const header = "timestamp,severity,requestId,functionName,message";
    const rows = filteredLogs.map(
      (e) =>
        `${e.timestamp},${e.severity},${e.requestId},${e.functionName ?? ""},"${e.message.replace(/"/g, '""')}"`
    );
    const csv = [header, ...rows].join("\n");
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `function-logs-${new Date().toISOString().slice(0, 10)}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const handleClear = () => {
    setLogs([]);
  };

  const projectBase = id ? `/projects/${id}` : "/dashboard";

  return (
    <div className="flex-1 min-h-0 flex flex-col">
      {/* Breadcrumbs */}
      <div className="flex items-center gap-2 text-sm text-base-content/70 mb-2">
        <Link to={projectBase} className="hover:text-base-content">
          Project
        </Link>
        <ChevronRight className="w-4 h-4 opacity-50" />
        <Link to={`${projectBase}/functions`} className="hover:text-base-content">
          Functions
        </Link>
        <ChevronRight className="w-4 h-4 opacity-50" />
        <span className="text-base-content font-medium">System Logs</span>
      </div>

      {/* Header */}
      <div className="flex flex-wrap items-start justify-between gap-4 mb-4">
        <div>
          <h1 className="text-2xl font-bold text-base-content">System Logs</h1>
          <p className="text-sm text-base-content/70 mt-0.5">
            Real-time execution logs for project {projectName}.
          </p>
        </div>
        <div className="flex items-center gap-2">
          {live && (
            <span className="badge badge-warning badge-lg gap-1">
              <span className="w-2 h-2 rounded-full bg-warning-content animate-pulse" />
              LIVE STREAM ACTIVE
            </span>
          )}
          {live && (
            <button
              type="button"
              className="btn btn-ghost btn-sm gap-1"
              onClick={() => setPaused((p) => !p)}
            >
              <Pause className="w-4 h-4" />
              {paused ? "Resume" : "Pause"}
            </button>
          )}
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-3 items-end mb-4">
        <div className="form-control">
          <label className="label py-0">
            <span className="label-text text-xs font-semibold uppercase">
              Severity
            </span>
          </label>
          <select
            className="select select-bordered select-sm w-40"
            value={severity}
            onChange={(e) => setSeverity(e.target.value)}
          >
            {SEVERITIES.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
        </div>
        <div className="form-control">
          <label className="label py-0">
            <span className="label-text text-xs font-semibold uppercase">
              Function
            </span>
          </label>
          <select
            className="select select-bordered select-sm w-44"
            value={functionFilter}
            onChange={(e) => setFunctionFilter(e.target.value)}
          >
            <option value="">All functions</option>
            {FUNCTION_NAMES.map((name) => (
              <option key={name} value={name}>
                {name}
              </option>
            ))}
          </select>
        </div>
        <div className="form-control">
          <label className="label py-0">
            <span className="label-text text-xs font-semibold uppercase">
              Time range
            </span>
          </label>
          <select
            className="select select-bordered select-sm w-40"
            value={timeRange}
            onChange={(e) => setTimeRange(e.target.value)}
          >
            {TIME_RANGES.map((r) => (
              <option key={r.value} value={r.value}>
                {r.label}
              </option>
            ))}
          </select>
        </div>
        <div className="form-control">
          <label className="label py-0">
            <span className="label-text text-xs font-semibold uppercase">
              Filter keywords
            </span>
          </label>
          <div className="relative">
            <Filter className="absolute left-2 top-1/2 -translate-y-1/2 w-4 h-4 text-base-content/50" />
            <input
              type="text"
              placeholder="Filter by Request ID, message..."
              className="input input-bordered input-sm pl-8 w-64"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
        </div>
        <button
          type="button"
          className="btn btn-ghost btn-sm gap-1"
          onClick={exportCsv}
        >
          <Download className="w-4 h-4" />
          CSV
        </button>
        <button
          type="button"
          className="btn btn-ghost btn-sm gap-1 text-error"
          onClick={handleClear}
        >
          <Trash2 className="w-4 h-4" />
          Clear
        </button>
      </div>

      {/* Log panel */}
      <div className="card bg-base-100 shadow-md flex-1 min-h-0 flex flex-col overflow-hidden border border-base-300">
        <div className="flex items-center justify-between px-4 py-2 border-b border-base-300 bg-base-200/50">
          <div className="flex items-center gap-2">
            <span className="w-3 h-3 rounded-full bg-error/80" />
            <span className="w-3 h-3 rounded-full bg-warning/80" />
            <span className="w-3 h-3 rounded-full bg-success/80" />
            <span className="text-sm font-mono text-base-content/70 ml-2">
              LOG-STREAM-TERMINAL-V1.0.4
            </span>
          </div>
        </div>
        <div
          ref={panelRef}
          className="flex-1 overflow-y-auto overflow-x-auto p-4 bg-base-300/30 font-mono text-sm min-h-[280px]"
        >
          {filteredLogs.length === 0 ? (
            <p className="text-base-content/50">No log entries match the current filters.</p>
          ) : (
            <ul className="space-y-1">
              {filteredLogs.map((entry) => (
                <li key={entry.id} className="leading-relaxed">
                  <span className="text-base-content/60 shrink-0 mr-2">
                    {formatTime(entry.timestamp)}
                  </span>
                  <span className={`font-semibold ${severityColor(entry.severity)}`}>
                    [{entry.severity}]
                  </span>
                  <span className="text-base-content/70 ml-2">{entry.requestId}</span>
                  <span className="ml-2 text-base-content/90">{entry.message}</span>
                  {entry.payload && (
                    <JsonBlock payload={entry.payload} />
                  )}
                </li>
              ))}
            </ul>
          )}
        </div>
        <div className="flex items-center justify-between px-4 py-2 border-t border-base-300 bg-base-200/50 text-xs text-base-content/60">
          <div className="flex items-center gap-4">
            <span>Buffer: 12.4 MB / 100 MB</span>
            <span>Stream: 24 msg/sec</span>
          </div>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              className="checkbox checkbox-sm"
              checked={autoScroll}
              onChange={(e) => setAutoScroll(e.target.checked)}
            />
            <span className="font-medium">AUTO-SCROLL</span>
          </label>
        </div>
      </div>
    </div>
  );
}
