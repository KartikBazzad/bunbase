import { useState, useEffect } from "react";
import { useAuthProviders } from "../../hooks/useAuthProviders";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../ui/card";
import { StatsCard } from "../ui/stats-card";
import { Loader2, Users, TrendingUp } from "lucide-react";
import { toast } from "sonner";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "../ui/chart";
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, ResponsiveContainer, PieChart, Pie, Cell } from "recharts";

interface ProviderStatisticsProps {
  projectId: string;
}

const COLORS = ["#0088FE", "#00C49F", "#FFBB28", "#FF8042", "#8884d8"];

export function ProviderStatistics({ projectId }: ProviderStatisticsProps) {
  const { getStatistics } = useAuthProviders(projectId);
  const [statistics, setStatistics] = useState<{
    totalUsers: number;
    providerBreakdown: Record<string, number>;
    signupsOverTime: Record<string, number>;
  } | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    loadStatistics();
  }, [projectId]);

  const loadStatistics = async () => {
    setIsLoading(true);
    try {
      const stats = await getStatistics();
      setStatistics(stats);
    } catch (error) {
      toast.error("Failed to load statistics");
    } finally {
      setIsLoading(false);
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!statistics) {
    return (
      <Card>
        <CardContent className="py-12">
          <p className="text-center text-muted-foreground">No statistics available</p>
        </CardContent>
      </Card>
    );
  }

  // Prepare data for charts
  const providerData = Object.entries(statistics.providerBreakdown || {}).map(([name, value]) => ({
    name: name.charAt(0).toUpperCase() + name.slice(1),
    value,
  }));

  const signupsData = Object.entries(statistics.signupsOverTime || {})
    .sort(([a], [b]) => a.localeCompare(b))
    .slice(-30) // Last 30 days
    .map(([date, count]) => ({
      date: new Date(date).toLocaleDateString("en-US", { month: "short", day: "numeric" }),
      count,
    }));

  const totalSignups = Object.values(statistics.signupsOverTime || {}).reduce(
    (sum, count) => sum + count,
    0,
  );

  return (
    <div className="space-y-6">
      <div className="grid gap-4 md:grid-cols-3">
        <StatsCard
          title="Total Users"
          value={statistics.totalUsers || 0}
          icon={Users}
          description="All registered users"
        />
        <StatsCard
          title="Total Signups"
          value={totalSignups}
          icon={TrendingUp}
          description="All time signups"
        />
        <StatsCard
          title="Active Providers"
          value={Object.keys(statistics.providerBreakdown || {}).length}
          description="Authentication methods in use"
        />
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Signups by Provider</CardTitle>
            <CardDescription>Distribution of user signups by authentication method</CardDescription>
          </CardHeader>
          <CardContent>
            {providerData.length > 0 ? (
              <ChartContainer
                config={{
                  count: {
                    label: "Users",
                  },
                }}
                className="h-[300px]"
              >
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={providerData}
                      cx="50%"
                      cy="50%"
                      labelLine={false}
                      label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                      outerRadius={80}
                      fill="#8884d8"
                      dataKey="value"
                    >
                      {providerData.map((entry, index) => (
                        <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                      ))}
                    </Pie>
                    <ChartTooltip content={<ChartTooltipContent />} />
                  </PieChart>
                </ResponsiveContainer>
              </ChartContainer>
            ) : (
              <div className="flex items-center justify-center h-[300px] text-muted-foreground">
                No data available
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Signups Over Time</CardTitle>
            <CardDescription>Daily signup trends (last 30 days)</CardDescription>
          </CardHeader>
          <CardContent>
            {signupsData.length > 0 ? (
              <ChartContainer
                config={{
                  count: {
                    label: "Signups",
                  },
                }}
                className="h-[300px]"
              >
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={signupsData}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis
                      dataKey="date"
                      tick={{ fontSize: 12 }}
                      angle={-45}
                      textAnchor="end"
                      height={80}
                    />
                    <YAxis tick={{ fontSize: 12 }} />
                    <ChartTooltip content={<ChartTooltipContent />} />
                    <Bar dataKey="count" fill="#0088FE" radius={[4, 4, 0, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              </ChartContainer>
            ) : (
              <div className="flex items-center justify-center h-[300px] text-muted-foreground">
                No data available
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Provider Breakdown</CardTitle>
          <CardDescription>Detailed statistics by authentication provider</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            {providerData.length > 0 ? (
              providerData.map((item, index) => (
                <div key={item.name} className="flex items-center justify-between p-3 rounded-lg border">
                  <div className="flex items-center gap-3">
                    <div
                      className="w-4 h-4 rounded"
                      style={{ backgroundColor: COLORS[index % COLORS.length] }}
                    />
                    <span className="font-medium">{item.name}</span>
                  </div>
                  <span className="text-muted-foreground">{item.value} users</span>
                </div>
              ))
            ) : (
              <p className="text-center text-muted-foreground py-4">No provider data available</p>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
