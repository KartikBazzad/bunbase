import { useState, useEffect } from "react";
import { useAuthProviders } from "../../hooks/useAuthProviders";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../ui/card";
import { Switch } from "../ui/switch";
import { Label } from "../ui/label";
import { Input } from "../ui/input";
import { Button } from "../ui/button";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";

interface EmailPasswordSettingsProps {
  projectId: string;
}

export function EmailPasswordSettings({ projectId }: EmailPasswordSettingsProps) {
  const { getAuthSettings, updateAuthSettings, isUpdatingSettings } = useAuthProviders(projectId);

  const [requireEmailVerification, setRequireEmailVerification] = useState(false);
  const [minPasswordLength, setMinPasswordLength] = useState(8);
  const [requireUppercase, setRequireUppercase] = useState(false);
  const [requireLowercase, setRequireLowercase] = useState(false);
  const [requireNumbers, setRequireNumbers] = useState(false);
  const [requireSpecialChars, setRequireSpecialChars] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    loadSettings();
  }, [projectId]);

  const loadSettings = async () => {
    setIsLoading(true);
    try {
      const settings = await getAuthSettings();
      if (settings) {
        setRequireEmailVerification(settings.requireEmailVerification || false);
        setMinPasswordLength(settings.minPasswordLength || 8);
        setRequireUppercase(settings.requireUppercase || false);
        setRequireLowercase(settings.requireLowercase || false);
        setRequireNumbers(settings.requireNumbers || false);
        setRequireSpecialChars(settings.requireSpecialChars || false);
      }
    } catch (error) {
      toast.error("Failed to load email/password settings");
    } finally {
      setIsLoading(false);
    }
  };

  const handleSave = async () => {
    try {
      await updateAuthSettings({
        requireEmailVerification,
        minPasswordLength,
        requireUppercase,
        requireLowercase,
        requireNumbers,
        requireSpecialChars,
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
        <CardTitle>Email & Password Settings</CardTitle>
        <CardDescription>
          Configure email verification and password requirements for user accounts
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label htmlFor="emailVerification">Require Email Verification</Label>
              <p className="text-sm text-muted-foreground">
                Users must verify their email address before they can sign in
              </p>
            </div>
            <Switch
              id="emailVerification"
              checked={requireEmailVerification}
              onCheckedChange={setRequireEmailVerification}
            />
          </div>

          <div className="space-y-4 border-t pt-4">
            <div>
              <Label className="text-base font-semibold">Password Requirements</Label>
              <p className="text-sm text-muted-foreground mt-1">
                Set minimum password complexity requirements
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="minPasswordLength">Minimum Password Length</Label>
              <Input
                id="minPasswordLength"
                type="number"
                min="4"
                max="128"
                value={minPasswordLength}
                onChange={(e) => setMinPasswordLength(parseInt(e.target.value) || 8)}
                className="w-32"
              />
              <p className="text-xs text-muted-foreground">
                Minimum number of characters required (4-128)
              </p>
            </div>

            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label htmlFor="requireUppercase">Require Uppercase Letter</Label>
                  <p className="text-sm text-muted-foreground">
                    Password must contain at least one uppercase letter (A-Z)
                  </p>
                </div>
                <Switch
                  id="requireUppercase"
                  checked={requireUppercase}
                  onCheckedChange={setRequireUppercase}
                />
              </div>

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label htmlFor="requireLowercase">Require Lowercase Letter</Label>
                  <p className="text-sm text-muted-foreground">
                    Password must contain at least one lowercase letter (a-z)
                  </p>
                </div>
                <Switch
                  id="requireLowercase"
                  checked={requireLowercase}
                  onCheckedChange={setRequireLowercase}
                />
              </div>

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label htmlFor="requireNumbers">Require Number</Label>
                  <p className="text-sm text-muted-foreground">
                    Password must contain at least one number (0-9)
                  </p>
                </div>
                <Switch
                  id="requireNumbers"
                  checked={requireNumbers}
                  onCheckedChange={setRequireNumbers}
                />
              </div>

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label htmlFor="requireSpecialChars">Require Special Character</Label>
                  <p className="text-sm text-muted-foreground">
                    Password must contain at least one special character (!@#$%^&*)
                  </p>
                </div>
                <Switch
                  id="requireSpecialChars"
                  checked={requireSpecialChars}
                  onCheckedChange={setRequireSpecialChars}
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
