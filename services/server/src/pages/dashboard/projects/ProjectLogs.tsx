import { useState, useMemo, useEffect } from "react";
import { useParams } from "react-router-dom";
import { useProject } from "../../../hooks/useProjects";
import {
  useProjectLogs,
  type ProjectLogRecord,
} from "../../../hooks/useProjectLogs";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../components/ui/card";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../../../components/ui/select";
import { Skeleton } from "../../../components/ui/skeleton";
import { LogStatsBar } from "../../../components/logs/LogStatsBar";
import { LogSplitView } from "../../../components/logs/LogSplitView";
import { ActiveFilters } from "../../../components/logs/FilterChips";
import { LiveModeToggle } from "../../../components/logs/RealtimeIndicator";
import { RefreshCw, Search, Filter, Download } from "lucide-react";
import { toast } from "sonner";

const LOG_LEVELS = ["all", "debug", "info", "warn", "error"] as const;
const LOG_TYPES = [
  "all",
  "project_operation",
  "database_operation",
  "storage_operation",
  "auth_event",
  "api_call",
] as const;

const PAGE_SIZES = [50, 100, 200, 500] as const;

export function ProjectLogs() {
  const { id } = useParams<{ id: string }>();
  const { project, isLoading: projectLoading } = useProject(id || "");

  const [levelFilter, setLevelFilter] = useState<string>("all");
  const [typeFilter, setTypeFilter] = useState<string>("all");
  const [searchQuery, setSearchQuery] = useState<string>("");
  const [pageSize, setPageSize] = useState<number>(50);
  const [offset, setOffset] = useState<number>(0);
  const [selectedLogId, setSelectedLogId] = useState<string | null>(null);
  const [isLive, setIsLive] = useState<boolean>(false);
  const [previousLogs, setPreviousLogs] = useState<ProjectLogRecord[]>([]);

  const logOptions = useMemo(
    () => ({
      limit: pageSize,
      offset,
      level:
        levelFilter !== "all"
          ? (levelFilter as "debug" | "info" | "warn" | "error")
          : undefined,
      type: typeFilter !== "all" ? typeFilter : undefined,
      search: searchQuery || undefined,
      full: true,
    }),
    [levelFilter, typeFilter, searchQuery, pageSize, offset],
  );

  const { logs, isLoading, error, refetch, hasMore } = useProjectLogs(
    id || "",
    logOptions,
  ) as {
    logs: ProjectLogRecord[];
    isLoading: boolean;
    error: Error | null;
    refetch: () => void;
    hasMore?: boolean;
  };

  useEffect(() => {
    setOffset(0);
    setSelectedLogId(null);
  }, [levelFilter, typeFilter, searchQuery, pageSize]);

  useEffect(() => {
    if (logs.length > 0 && !selectedLogId) {
      setSelectedLogId(logs[0]?.id ?? null);
    }
  }, [logs]);

  useEffect(() => {
    if (isLive && !isLoading) {
      const interval = setInterval(() => {
        refetch();
      }, 5000);
      return () => clearInterval(interval);
    }
  }, [isLive, isLoading, refetch]);

  useEffect(() => {
    if (logs.length > 0 && previousLogs.length === 0) {
      setPreviousLogs([...logs]);
    }
  }, [logs.length]);

  const clearFilters = () => {
    setLevelFilter("all");
    setTypeFilter("all");
    setSearchQuery("");
    setOffset(0);
  };

  const loadMore = () => {
    setOffset((prev) => prev + pageSize);
  };

  const exportLogs = async () => {
    try {
      const exportData = logs.map((log) => ({
        id: log.id,
        level: log.level,
        message: log.message,
        timestamp: log.timestamp.toISOString(),
        context: log.context,
        metadata: log.metadata,
        correlationId: log.correlationId,
        service: log.service,
        type: log.type,
      }));

      const blob = new Blob([JSON.stringify(exportData, null, 2)], {
        type: "application/json",
      });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `logs-${new Date().toISOString()}.json`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
      toast.success("Logs exported successfully");
    } catch (error) {
      toast.error("Failed to export logs");
    }
  };

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (
        e.target instanceof HTMLInputElement ||
        e.target instanceof HTMLTextAreaElement
      )
        return;

      const currentIndex = logs.findIndex((log) => log.id === selectedLogId);

      switch (e.key) {
        case "ArrowDown":
        case "j":
          e.preventDefault();
          const nextLog = logs[currentIndex + 1];
          if (currentIndex < logs.length - 1 && nextLog) {
            setSelectedLogId(nextLog.id);
          }
          break;
        case "ArrowUp":
        case "k":
          e.preventDefault();
          const prevLog = logs[currentIndex - 1];
          if (currentIndex > 0 && prevLog) {
            setSelectedLogId(prevLog.id);
          }
          break;
        case "Escape":
          setSelectedLogId(null);
          break;
        case "l":
          setIsLive((prev) => !prev);
          break;
        case "r":
          e.preventDefault();
          refetch();
          break;
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [logs, selectedLogId, refetch]);

  const hasActiveFilters =
    levelFilter !== "all" || typeFilter !== "all" || searchQuery !== "";

  if (projectLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-32 w-full" />
      </div>
    );
  }

  if (!project) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-center">
          <h2 className="text-2xl font-bold">Project not found</h2>
          <p className="text-muted-foreground mt-2">
            The project you're looking for doesn't exist or you don't have
            access to it.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="space-y-1">
          <h1 className="text-3xl font-bold tracking-tight">Logs</h1>
          <p className="text-muted-foreground">
            View and filter project logs for {project.name}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <LiveModeToggle isLive={isLive} onToggle={() => setIsLive(!isLive)} />
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCw
              className={`h-4 w-4 mr-2 ${isLoading ? "animate-spin" : ""}`}
            />
            Refresh
          </Button>
          <Button variant="outline" size="sm" onClick={exportLogs}>
            <Download className="h-4 w-4 mr-2" />
            Export
          </Button>
        </div>
      </div>

      <LogStatsBar logs={logs} previousLogs={previousLogs} />

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Filter className="h-4 w-4" />
              <CardTitle>Filters</CardTitle>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-wrap gap-4">
            <Select value={levelFilter} onValueChange={setLevelFilter}>
              <SelectTrigger className="w-[180px]">
                <SelectValue placeholder="All levels" />
              </SelectTrigger>
              <SelectContent>
                {LOG_LEVELS.map((level) => (
                  <SelectItem key={level} value={level}>
                    {level === "all" ? "All Levels" : level.toUpperCase()}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            <Select value={typeFilter} onValueChange={setTypeFilter}>
              <SelectTrigger className="w-[200px]">
                <SelectValue placeholder="All types" />
              </SelectTrigger>
              <SelectContent>
                {LOG_TYPES.map((type) => (
                  <SelectItem key={type} value={type}>
                    {type === "all"
                      ? "All Types"
                      : type
                          .replace(/_/g, " ")
                          .replace(/\b\w/g, (c) => c.toUpperCase())}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            <Select
              value={pageSize.toString()}
              onValueChange={(value) => {
                setPageSize(parseInt(value));
                setOffset(0);
              }}
            >
              <SelectTrigger className="w-[120px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {PAGE_SIZES.map((size) => (
                  <SelectItem key={size} value={size.toString()}>
                    {size}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            <div className="relative flex-1 min-w-[200px]">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search messages..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9"
              />
            </div>
          </div>

          <ActiveFilters
            levelFilter={levelFilter}
            typeFilter={typeFilter}
            searchQuery={searchQuery}
            onClearLevel={() => setLevelFilter("all")}
            onClearType={() => setTypeFilter("all")}
            onClearSearch={() => setSearchQuery("")}
            onClearAll={clearFilters}
            logCount={logs.length}
          />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Log Entries</CardTitle>
            <div className="flex items-center gap-4 text-xs text-muted-foreground">
              <span>↑↓/jk Navigate</span>
              <span>l Toggle Live</span>
              <span>r Refresh</span>
            </div>
          </div>
        </CardHeader>
        <CardContent className="p-0">
          <LogSplitView
            logs={logs}
            selectedLogId={selectedLogId}
            onSelectLog={setSelectedLogId}
            onLoadMore={loadMore}
            hasMore={hasMore}
            isLoading={isLoading}
          />
        </CardContent>
      </Card>
    </div>
  );
}
