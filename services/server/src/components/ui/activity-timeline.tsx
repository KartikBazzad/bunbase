import { Card, CardContent } from "./card";
import { cn } from "@/lib/utils";
import type { LucideIcon } from "lucide-react";

export interface ActivityItem {
  id: string;
  title: string;
  description?: string;
  icon?: LucideIcon;
  timestamp: Date;
  type?: "success" | "warning" | "error" | "info";
}

interface ActivityTimelineProps {
  activities: ActivityItem[];
  className?: string;
}

export function ActivityTimeline({ activities, className }: ActivityTimelineProps) {
  const getIconBgColor = (type?: string) => {
    switch (type) {
      case "success":
        return "bg-green-100 dark:bg-green-900/20 text-green-600 dark:text-green-500";
      case "warning":
        return "bg-yellow-100 dark:bg-yellow-900/20 text-yellow-600 dark:text-yellow-500";
      case "error":
        return "bg-red-100 dark:bg-red-900/20 text-red-600 dark:text-red-500";
      default:
        return "bg-blue-100 dark:bg-blue-900/20 text-blue-600 dark:text-blue-500";
    }
  };

  const formatTimestamp = (date: Date) => {
    const now = new Date();
    const diffInMs = now.getTime() - date.getTime();
    const diffInMins = Math.floor(diffInMs / 60000);
    const diffInHours = Math.floor(diffInMs / 3600000);
    const diffInDays = Math.floor(diffInMs / 86400000);

    if (diffInMins < 1) return "Just now";
    if (diffInMins < 60) return `${diffInMins}m ago`;
    if (diffInHours < 24) return `${diffInHours}h ago`;
    if (diffInDays < 7) return `${diffInDays}d ago`;
    return date.toLocaleDateString();
  };

  return (
    <Card className={cn("", className)}>
      <CardContent className="p-0">
        <div className="divide-y divide-border/50">
          {activities.length === 0 ? (
            <div className="p-8 text-center text-sm text-muted-foreground">
              No recent activity
            </div>
          ) : (
            activities.map((activity, index) => {
              const Icon = activity.icon;
              return (
                <div
                  key={activity.id}
                  className={cn(
                    "group relative flex gap-3 p-4 transition-colors hover:bg-muted/30",
                    index === 0 && "pt-6"
                  )}
                >
                  <div className="relative flex shrink-0">
                    <div
                      className={cn(
                        "flex size-8 items-center justify-center rounded-md",
                        getIconBgColor(activity.type)
                      )}
                    >
                      {Icon ? <Icon className="size-4" /> : null}
                    </div>
                    {index !== activities.length - 1 && (
                      <div className="absolute left-4 top-8 h-full w-px bg-border/50" />
                    )}
                  </div>
                  <div className="flex min-w-0 flex-1 flex-col gap-1">
                    <div className="flex items-center gap-2">
                      <p className="text-sm font-medium">{activity.title}</p>
                      <span className="text-xs text-muted-foreground">
                        {formatTimestamp(activity.timestamp)}
                      </span>
                    </div>
                    {activity.description && (
                      <p className="text-xs text-muted-foreground">
                        {activity.description}
                      </p>
                    )}
                  </div>
                </div>
              );
            })
          )}
        </div>
      </CardContent>
    </Card>
  );
}
