/**
 * Dummy data and merge helper for Functions and Logs UI.
 * Used when API returns empty or to fill extended fields (trigger, status, etc.).
 */

export type TriggerType = "http" | "database" | "scheduled";
export type FunctionStatus = "active" | "deploying" | "failed";

export interface FunctionRow {
  id: string;
  name: string;
  runtime: string;
  trigger: TriggerType;
  pathOrCron: string;
  status: FunctionStatus;
  lastDeployed: string;
  /** From API when merged */
  project_id?: string;
  function_service_id?: string;
  created_at?: string;
  updated_at?: string;
}

export interface DummyMetrics {
  totalFunctions: number;
  invocations24h: string;
  invocationsTrend: string;
  invocationsSubtitle: string;
  errorRate: string;
  errorRateTrend: string;
  errorCountSubtitle: string;
}

export interface LogEntry {
  id: string;
  timestamp: string; // ISO or display
  severity: "INFO" | "WARN" | "ERROR" | "DEBUG";
  requestId: string;
  message: string;
  functionName?: string;
  payload?: string; // JSON string for syntax-highlighted block
}

/** Extended function rows for table when API returns nothing */
export const DUMMY_FUNCTION_ROWS: FunctionRow[] = [
  {
    id: "fn-1",
    name: "stripe-webhook-handler",
    runtime: "Bun 1.1.2",
    trigger: "http",
    pathOrCron: "/v1/webhooks/stripe",
    status: "active",
    lastDeployed: "2 mins ago",
  },
  {
    id: "fn-2",
    name: "auth-cleanup-job",
    runtime: "Node.js 20",
    trigger: "scheduled",
    pathOrCron: "cron: 0 0 * * *",
    status: "active",
    lastDeployed: "1 hour ago",
  },
  {
    id: "fn-3",
    name: "image-processor-worker",
    runtime: "Node.js 18",
    trigger: "database",
    pathOrCron: "bucket: uploads-prod",
    status: "deploying",
    lastDeployed: "5 mins ago",
  },
  {
    id: "fn-4",
    name: "email-notification-service",
    runtime: "Bun 1.1.2",
    trigger: "http",
    pathOrCron: "/v1/notify/email",
    status: "failed",
    lastDeployed: "Yesterday",
  },
];

export const DUMMY_METRICS: DummyMetrics = {
  totalFunctions: 12,
  invocations24h: "45.2k",
  invocationsTrend: "+15%",
  invocationsSubtitle: "Avg 1.8k per hour",
  errorRate: "0.02%",
  errorRateTrend: "-5%",
  errorCountSubtitle: "12 failed executions",
};

const STRIPE_PAYLOAD = `{
  "event": "payment.succeeded",
  "customer_id": "cus_99182",
  "amount": 4999
}`;

/** Base time for dummy logs (recent) */
function recentTime(minutesAgo: number): string {
  const d = new Date(Date.now() - minutesAgo * 60 * 1000);
  return d.toISOString();
}

export const DUMMY_LOG_ENTRIES: LogEntry[] = [
  {
    id: "log-1",
    timestamp: recentTime(2),
    severity: "INFO",
    requestId: "REQ-88214A",
    message: "Initializing user-auth-hook execution environment...",
    functionName: "user-auth-hook",
  },
  {
    id: "log-2",
    timestamp: recentTime(3),
    severity: "WARN",
    requestId: "REQ-88214A",
    message: "Memory usage reaching 85% limit for isolate 0x3a4f.",
    functionName: "user-auth-hook",
  },
  {
    id: "log-3",
    timestamp: recentTime(4),
    severity: "ERROR",
    requestId: "REQ-771B2C",
    message: "DatabaseConnectionError: Pool size limit exceeded. View Stacktrace",
    functionName: "stripe-webhook-handler",
  },
  {
    id: "log-4",
    timestamp: recentTime(5),
    severity: "INFO",
    requestId: "REQ-99182D",
    message: "Incoming Webhook Payload:",
    functionName: "stripe-webhook-handler",
    payload: STRIPE_PAYLOAD,
  },
  {
    id: "log-5",
    timestamp: recentTime(6),
    severity: "INFO",
    requestId: "REQ-551A3E",
    message: "Health check: Status OK - Nodes: 12",
    functionName: "auth-cleanup-job",
  },
  {
    id: "log-6",
    timestamp: recentTime(8),
    severity: "INFO",
    requestId: "REQ-662B4F",
    message: "Image resize started for /assets/banner_v2.png",
    functionName: "image-processor-worker",
  },
  {
    id: "log-7",
    timestamp: recentTime(10),
    severity: "DEBUG",
    requestId: "REQ-773C5A",
    message: "Cache hit for key sessions:user:abc123",
    functionName: "user-auth-hook",
  },
  {
    id: "log-8",
    timestamp: recentTime(12),
    severity: "INFO",
    requestId: "REQ-884D6B",
    message: "Cron trigger fired: 0 0 * * *",
    functionName: "auth-cleanup-job",
  },
  {
    id: "log-9",
    timestamp: recentTime(14),
    severity: "WARN",
    requestId: "REQ-995E7C",
    message: "Retry attempt 2/3 for external API call",
    functionName: "email-notification-service",
  },
  {
    id: "log-10",
    timestamp: recentTime(15),
    severity: "INFO",
    requestId: "REQ-AA6F8D",
    message: "_ Waiting for new events...",
    functionName: "user-auth-hook",
  },
  {
    id: "log-11",
    timestamp: recentTime(1),
    severity: "INFO",
    requestId: "REQ-BB7A9E",
    message: "Function cold start completed in 120ms",
    functionName: "stripe-webhook-handler",
  },
  {
    id: "log-12",
    timestamp: recentTime(7),
    severity: "ERROR",
    requestId: "REQ-CC8B0F",
    message: "SMTP connection timeout after 5000ms",
    functionName: "email-notification-service",
  },
  {
    id: "log-13",
    timestamp: recentTime(9),
    severity: "INFO",
    requestId: "REQ-DD9C1A",
    message: "Document change detected on collection uploads",
    functionName: "image-processor-worker",
  },
  {
    id: "log-14",
    timestamp: recentTime(11),
    severity: "INFO",
    requestId: "REQ-EE0D2B",
    message: "Token validation passed for REQ-88214A",
    functionName: "user-auth-hook",
  },
  {
    id: "log-15",
    timestamp: recentTime(13),
    severity: "WARN",
    requestId: "REQ-FF1E3C",
    message: "Deprecated API usage: consider migrating to v2",
    functionName: "auth-cleanup-job",
  },
];

/** API function shape from listFunctions (enriched with trigger, status, path_or_cron) */
export interface ApiFunction {
  id: string;
  project_id: string;
  function_service_id: string;
  name: string;
  runtime: string;
  trigger: TriggerType;
  status: FunctionStatus;
  path_or_cron: string;
  created_at: string;
  updated_at: string;
}

export function formatLastDeployed(updated_at: string): string {
  const d = new Date(updated_at);
  const now = Date.now();
  const diffMs = now - d.getTime();
  const diffMins = Math.floor(diffMs / 60_000);
  const diffHours = Math.floor(diffMins / 60);
  const diffDays = Math.floor(diffHours / 24);
  if (diffMins < 60) return `${diffMins} mins ago`;
  if (diffHours < 24) return `${diffHours} hour${diffHours === 1 ? "" : "s"} ago`;
  if (diffDays === 1) return "Yesterday";
  if (diffDays < 7) return `${diffDays} days ago`;
  return d.toLocaleDateString();
}

/**
 * Merge API list with dummy extended fields. If apiList is empty, return full dummy list.
 * @deprecated The API now returns enriched data. This function is kept for backward compatibility
 * but should not be used for new code. Use apiList directly instead.
 */
export function mergeFunctionsWithDummy(apiList: ApiFunction[]): FunctionRow[] {
  if (!apiList || apiList.length === 0) {
    return DUMMY_FUNCTION_ROWS.map((row) => ({ ...row }));
  }
  // API now returns enriched data, so we can map directly
  return apiList.map((fn) => {
    return {
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
    };
  });
}
