import { useState, useEffect } from "react";
import { useAuthProviders } from "../../hooks/useAuthProviders";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../ui/card";
import { Switch } from "../ui/switch";
import { Label } from "../ui/label";
import { Input } from "../ui/input";
import { Button } from "../ui/button";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";

interface AdvancedAuthSettingsProps {
  projectId: string;
}

export function AdvancedAuthSettings({ projectId }: AdvancedAuthSettingsProps) {
  const { getAuthSettings, updateAuthSettings, isUpdatingSettings } = useAuthProviders(projectId);

  const [sessionExpirationDays, setSessionExpirationDays] = useState(30);
  const [rateLimitMax, setRateLimitMax] = useState(5);
  const [rateLimitWindow, setRateLimitWindow] = useState(15);
  const [mfaEnabled, setMfaEnabled] = useState(false);
  const [mfaRequired, setMfaRequired] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    loadSettings();
  }, [projectId]);

  const loadSettings = async () => {
    setIsLoading(true);
    try {
      const settings = await getAuthSettings();
      if (settings) {
        setSessionExpirationDays(settings.sessionExpirationDays || 30);
        setRateLimitMax(settings.rateLimitMax || 5);
        setRateLimitWindow(settings.rateLimitWindow || 15);
        setMfaEnabled(settings.mfaEnabled || false);
        setMfaRequired(settings.mfaRequired || false);
      }
    } catch (error) {
      toast.error("Failed to load advanced settings");
    } finally {
      setIsLoading(false);
    }
  };

  const handleSave = async () => {
    try {
      await updateAuthSettings({
        sessionExpirationDays,
        rateLimitMax,
        rateLimitWindow,
        mfaEnabled,
        mfaRequired,
      });
    } catch (error) {
      // Error is handled by the hook
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Advanced Settings</CardTitle>
        <CardDescription>
          Configure session management, rate limiting, and multi-factor authentication
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="space-y-4">
          <div className="space-y-4 border-b pb-4">
            <div>
              <Label className="text-base font-semibold">Session Management</Label>
              <p className="text-sm text-muted-foreground mt-1">
                Control how long user sessions remain active
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="sessionExpiration">Session Expiration (Days)</Label>
              <Input
                id="sessionExpiration"
                type="number"
                min="1"
                max="365"
                value={sessionExpirationDays}
                onChange={(e) => setSessionExpirationDays(parseInt(e.target.value) || 30)}
                className="w-32"
              />
              <p className="text-xs text-muted-foreground">
                Number of days before a session expires (1-365)
              </p>
            </div>
          </div>

          <div className="space-y-4 border-b pb-4">
            <div>
              <Label className="text-base font-semibold">Rate Limiting</Label>
              <p className="text-sm text-muted-foreground mt-1">
                Protect against brute force attacks by limiting authentication attempts
              </p>
            </div>

            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="rateLimitMax">Max Attempts</Label>
                <Input
                  id="rateLimitMax"
                  type="number"
                  min="1"
                  max="100"
                  value={rateLimitMax}
                  onChange={(e) => setRateLimitMax(parseInt(e.target.value) || 5)}
                  className="w-32"
                />
                <p className="text-xs text-muted-foreground">
                  Maximum number of failed attempts allowed
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="rateLimitWindow">Time Window (Minutes)</Label>
                <Input
                  id="rateLimitWindow"
                  type="number"
                  min="1"
                  max="1440"
                  value={rateLimitWindow}
                  onChange={(e) => setRateLimitWindow(parseInt(e.target.value) || 15)}
                  className="w-32"
                />
                <p className="text-xs text-muted-foreground">
                  Time window in minutes for rate limiting (1-1440)
                </p>
              </div>
            </div>
          </div>

          <div className="space-y-4">
            <div>
              <Label className="text-base font-semibold">Multi-Factor Authentication (MFA)</Label>
              <p className="text-sm text-muted-foreground mt-1">
                Add an extra layer of security with two-factor authentication
              </p>
            </div>

            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label htmlFor="mfaEnabled">Enable MFA</Label>
                  <p className="text-sm text-muted-foreground">
                    Allow users to enable two-factor authentication for their accounts
                  </p>
                </div>
                <Switch id="mfaEnabled" checked={mfaEnabled} onCheckedChange={setMfaEnabled} />
              </div>

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label htmlFor="mfaRequired">Require MFA</Label>
                  <p className="text-sm text-muted-foreground">
                    Force all users to enable MFA (requires MFA to be enabled)
                  </p>
                </div>
                <Switch
                  id="mfaRequired"
                  checked={mfaRequired}
                  onCheckedChange={setMfaRequired}
                  disabled={!mfaEnabled}
                />
              </div>
            </div>
          </div>
        </div>

        <div className="flex justify-end pt-4 border-t">
          <Button onClick={handleSave} disabled={isUpdatingSettings}>
            {isUpdatingSettings ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Saving...
              </>
            ) : (
              "Save Settings"
            )}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
