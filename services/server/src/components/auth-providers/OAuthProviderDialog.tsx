import { useState, useEffect } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../ui/dialog";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { Label } from "../ui/label";
import { Checkbox } from "../ui/checkbox";
import { useAuthProviders } from "../../hooks/useAuthProviders";
import { Loader2, CheckCircle2, XCircle } from "lucide-react";
import { toast } from "sonner";

interface OAuthProviderDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  projectId: string;
  provider: "google" | "github" | "facebook" | "apple";
  providerLabel: string;
}

const defaultScopes: Record<string, string[]> = {
  google: ["openid", "email", "profile"],
  github: ["user:email"],
  facebook: ["email", "public_profile"],
  apple: ["email", "name"],
};

export function OAuthProviderDialog({
  open,
  onOpenChange,
  projectId,
  provider,
  providerLabel,
}: OAuthProviderDialogProps) {
  const { getOAuthConfig, saveOAuthConfig, testOAuthConnection, isSavingOAuth, isTestingOAuth } =
    useAuthProviders(projectId);

  const [clientId, setClientId] = useState("");
  const [clientSecret, setClientSecret] = useState("");
  const [redirectUri, setRedirectUri] = useState("");
  const [scopes, setScopes] = useState<string[]>(defaultScopes[provider] || []);
  const [testStatus, setTestStatus] = useState<"success" | "failed" | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    if (open && projectId) {
      loadConfig();
    }
  }, [open, projectId, provider]);

  const loadConfig = async () => {
    setIsLoading(true);
    try {
      const config = await getOAuthConfig(provider);
      if (config) {
        setClientId(config.clientId || "");
        setClientSecret(config.clientSecret || ""); // This will be masked
        setRedirectUri(config.redirectUri || "");
        setScopes((config.scopes as string[]) || defaultScopes[provider] || []);
        setTestStatus(config.lastTestStatus as "success" | "failed" | null);
      } else {
        // Reset to defaults
        setClientId("");
        setClientSecret("");
        setRedirectUri("");
        setScopes(defaultScopes[provider] || []);
        setTestStatus(null);
      }
    } catch (error) {
      toast.error("Failed to load OAuth configuration");
    } finally {
      setIsLoading(false);
    }
  };

  const handleScopeToggle = (scope: string) => {
    setScopes((prev) =>
      prev.includes(scope) ? prev.filter((s) => s !== scope) : [...prev, scope],
    );
  };

  const handleTest = async () => {
    if (!clientId || !clientSecret) {
      toast.error("Please fill in Client ID and Client Secret first");
      return;
    }

    try {
      // Save first, then test
      await handleSave(true);
      const result = await testOAuthConnection(provider);
      if (result?.success) {
        setTestStatus("success");
        toast.success("Connection test successful!");
      } else {
        setTestStatus("failed");
        toast.error(result?.message || "Connection test failed");
      }
    } catch (error) {
      setTestStatus("failed");
      toast.error("Connection test failed");
    }
  };

  const handleSave = async (skipToast = false) => {
    if (!clientId || !clientSecret) {
      toast.error("Client ID and Client Secret are required");
      return;
    }

    try {
      await saveOAuthConfig({
        provider,
        clientId,
        clientSecret,
        redirectUri: redirectUri || undefined,
        scopes,
      });
      if (!skipToast) {
        onOpenChange(false);
      }
    } catch (error) {
      // Error is handled by the hook
    }
  };

  const handleCancel = () => {
    setClientId("");
    setClientSecret("");
    setRedirectUri("");
    setScopes(defaultScopes[provider] || []);
    setTestStatus(null);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>Configure {providerLabel} OAuth</DialogTitle>
          <DialogDescription>
            Enter your OAuth application credentials. Your client secret will be encrypted and stored
            securely.
          </DialogDescription>
        </DialogHeader>

        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        ) : (
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="clientId">Client ID</Label>
              <Input
                id="clientId"
                value={clientId}
                onChange={(e) => setClientId(e.target.value)}
                placeholder="Enter your Client ID"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="clientSecret">Client Secret</Label>
              <Input
                id="clientSecret"
                type="password"
                value={clientSecret}
                onChange={(e) => setClientSecret(e.target.value)}
                placeholder="Enter your Client Secret"
              />
              {clientSecret && clientSecret.includes("*") && (
                <p className="text-xs text-muted-foreground">
                  Enter a new secret to update the existing one
                </p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="redirectUri">Redirect URI (Optional)</Label>
              <Input
                id="redirectUri"
                value={redirectUri}
                onChange={(e) => setRedirectUri(e.target.value)}
                placeholder="https://your-app.com/auth/callback"
              />
              <p className="text-xs text-muted-foreground">
                The callback URL registered with your OAuth provider
              </p>
            </div>

            <div className="space-y-2">
              <Label>Scopes</Label>
              <div className="space-y-2 rounded-md border p-3">
                {(defaultScopes[provider] || []).map((scope) => (
                  <div key={scope} className="flex items-center space-x-2">
                    <Checkbox
                      id={`scope-${scope}`}
                      checked={scopes.includes(scope)}
                      onCheckedChange={() => handleScopeToggle(scope)}
                    />
                    <Label
                      htmlFor={`scope-${scope}`}
                      className="text-sm font-normal cursor-pointer"
                    >
                      {scope}
                    </Label>
                  </div>
                ))}
              </div>
            </div>

            {testStatus && (
              <div
                className={`flex items-center gap-2 rounded-md border p-3 ${
                  testStatus === "success"
                    ? "bg-green-50 dark:bg-green-950 border-green-200 dark:border-green-800"
                    : "bg-red-50 dark:bg-red-950 border-red-200 dark:border-red-800"
                }`}
              >
                {testStatus === "success" ? (
                  <CheckCircle2 className="h-4 w-4 text-green-600 dark:text-green-400" />
                ) : (
                  <XCircle className="h-4 w-4 text-red-600 dark:text-red-400" />
                )}
                <span
                  className={`text-sm ${
                    testStatus === "success"
                      ? "text-green-700 dark:text-green-300"
                      : "text-red-700 dark:text-red-300"
                  }`}
                >
                  Last test: {testStatus === "success" ? "Success" : "Failed"}
                </span>
              </div>
            )}
          </div>
        )}

        <DialogFooter className="gap-2">
          <Button variant="outline" onClick={handleCancel} disabled={isSavingOAuth}>
            Cancel
          </Button>
          <Button
            variant="outline"
            onClick={handleTest}
            disabled={isSavingOAuth || isTestingOAuth || !clientId || !clientSecret}
          >
            {isTestingOAuth ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Testing...
              </>
            ) : (
              "Test Connection"
            )}
          </Button>
          <Button onClick={() => handleSave()} disabled={isSavingOAuth || !clientId || !clientSecret}>
            {isSavingOAuth ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Saving...
              </>
            ) : (
              "Save"
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
