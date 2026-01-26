import { useState, useEffect } from "react";
import { useAuthProviders } from "../../hooks/useAuthProviders";
import { Button } from "../ui/button";
import {
  Loader2,
  Mail,
  Github,
  Chrome,
  Facebook,
  Apple,
  Settings,
  CheckCircle2,
  XCircle,
  AlertCircle,
} from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../ui/card";
import { Switch } from "../ui/switch";
import { Label } from "../ui/label";
import { Badge } from "../ui/badge";
import { toast } from "sonner";
import { OAuthProviderDialog } from "./OAuthProviderDialog";

interface AuthProviderConfigProps {
  projectId: string;
}

const providerIcons = {
  email: Mail,
  google: Chrome,
  github: Github,
  facebook: Facebook,
  apple: Apple,
};

const providerLabels = {
  email: "Email",
  google: "Google",
  github: "GitHub",
  facebook: "Facebook",
  apple: "Apple",
};

const oauthProviders = ["google", "github", "facebook", "apple"];

export function AuthProviderConfig({ projectId }: AuthProviderConfigProps) {
  const {
    authConfig,
    isLoading,
    toggleProvider,
    isToggling,
    getOAuthConfig,
    testOAuthConnection,
    isTestingOAuth,
  } = useAuthProviders(projectId);

  const [configuringProvider, setConfiguringProvider] = useState<
    "google" | "github" | "facebook" | "apple" | null
  >(null);
  const [oauthConfigs, setOauthConfigs] = useState<
    Record<string, { isConfigured: boolean; lastTestStatus: string | null }>
  >({});
  const [loadingConfigs, setLoadingConfigs] = useState(false);

  useEffect(() => {
    if (projectId && !isLoading) {
      loadOAuthConfigs();
    }
  }, [projectId, isLoading]);

  const loadOAuthConfigs = async () => {
    setLoadingConfigs(true);
    const configs: Record<string, { isConfigured: boolean; lastTestStatus: string | null }> = {};
    for (const provider of oauthProviders) {
      try {
        const config = await getOAuthConfig(provider);
        if (config) {
          configs[provider] = {
            isConfigured: config.isConfigured || false,
            lastTestStatus: config.lastTestStatus || null,
          };
        }
      } catch (error) {
        // Provider not configured
      }
    }
    setOauthConfigs(configs);
    setLoadingConfigs(false);
  };

  const handleToggle = async (provider: string) => {
    if (provider === "email") {
      toast.error("Email authentication cannot be disabled");
      return;
    }

    await toggleProvider(provider);
    // Reload OAuth configs after toggle
    if (oauthProviders.includes(provider)) {
      await loadOAuthConfigs();
    }
  };

  const getProviderStatus = (provider: string) => {
    if (provider === "email") return null;
    if (!oauthProviders.includes(provider)) return null;

    const config = oauthConfigs[provider];
    if (!config) return "not-configured";

    if (config.lastTestStatus === "success") return "configured";
    if (config.lastTestStatus === "failed") return "failed";
    if (config.isConfigured) return "configured-no-test";
    return "not-configured";
  };

  const getStatusBadge = (status: string | null) => {
    if (!status) return null;

    switch (status) {
      case "configured":
        return (
          <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200">
            <CheckCircle2 className="h-3 w-3 mr-1" />
            Configured
          </Badge>
        );
      case "configured-no-test":
        return (
          <Badge variant="outline" className="bg-yellow-50 text-yellow-700 border-yellow-200">
            <AlertCircle className="h-3 w-3 mr-1" />
            Not Tested
          </Badge>
        );
      case "failed":
        return (
          <Badge variant="outline" className="bg-red-50 text-red-700 border-red-200">
            <XCircle className="h-3 w-3 mr-1" />
            Test Failed
          </Badge>
        );
      case "not-configured":
        return (
          <Badge variant="outline" className="bg-gray-50 text-gray-700 border-gray-200">
            Not Configured
          </Badge>
        );
      default:
        return null;
    }
  };

  if (isLoading || loadingConfigs) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  const enabledProviders = authConfig?.providers || ["email"];

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle>Authentication Providers</CardTitle>
          <CardDescription>
            Configure which authentication methods are available for this project
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {Object.entries(providerLabels).map(([key, label]) => {
            const Icon = providerIcons[key as keyof typeof providerIcons];
            const isEnabled = enabledProviders.includes(key);
            const isEmail = key === "email";
            const isOAuth = oauthProviders.includes(key);
            const status = getProviderStatus(key);

            return (
              <div
                key={key}
                className="flex items-center justify-between rounded-lg border p-4"
              >
                <div className="flex items-center gap-3 flex-1">
                  <Icon className="h-5 w-5 text-muted-foreground" />
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <Label htmlFor={key} className="font-medium">
                        {label}
                      </Label>
                      {getStatusBadge(status)}
                    </div>
                    {isEmail && (
                      <p className="text-xs text-muted-foreground">Always enabled</p>
                    )}
                    {isOAuth && !isEmail && (
                      <p className="text-xs text-muted-foreground">
                        {status === "not-configured"
                          ? "Configure OAuth credentials to enable"
                          : status === "failed"
                            ? "Connection test failed - check your credentials"
                            : "OAuth provider ready"}
                      </p>
                    )}
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  {isOAuth && !isEmail && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() =>
                        setConfiguringProvider(key as "google" | "github" | "facebook" | "apple")
                      }
                    >
                      <Settings className="h-4 w-4 mr-1" />
                      Configure
                    </Button>
                  )}
                  <Switch
                    id={key}
                    checked={isEnabled}
                    onCheckedChange={() => handleToggle(key)}
                    disabled={isEmail || isToggling}
                  />
                </div>
              </div>
            );
          })}
        </CardContent>
      </Card>

      {configuringProvider && (
        <OAuthProviderDialog
          open={!!configuringProvider}
          onOpenChange={(open) => {
            if (!open) {
              setConfiguringProvider(null);
              loadOAuthConfigs();
            }
          }}
          projectId={projectId}
          provider={configuringProvider}
          providerLabel={providerLabels[configuringProvider]}
        />
      )}
    </>
  );
}
