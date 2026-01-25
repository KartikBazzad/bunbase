import { useAuthProviders } from "../../hooks/useAuthProviders";
import { Button } from "../ui/button";
import { Loader2, Mail, Github, Chrome, Facebook, Apple } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../ui/card";
import { Switch } from "../ui/switch";
import { Label } from "../ui/label";
import { toast } from "sonner";

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

export function AuthProviderConfig({ projectId }: AuthProviderConfigProps) {
  const { authConfig, isLoading, toggleProvider, isToggling } =
    useAuthProviders(projectId);

  const handleToggle = async (provider: string) => {
    if (provider === "email") {
      toast.error("Email authentication cannot be disabled");
      return;
    }

    await toggleProvider(provider);
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  const enabledProviders = authConfig?.providers || ["email"];

  return (
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

          return (
            <div
              key={key}
              className="flex items-center justify-between rounded-lg border p-4"
            >
              <div className="flex items-center gap-3">
                <Icon className="h-5 w-5 text-muted-foreground" />
                <div>
                  <Label htmlFor={key} className="font-medium">
                    {label}
                  </Label>
                  {isEmail && (
                    <p className="text-xs text-muted-foreground">
                      Always enabled
                    </p>
                  )}
                </div>
              </div>
              <Switch
                id={key}
                checked={isEnabled}
                onCheckedChange={() => handleToggle(key)}
                disabled={isEmail || isToggling}
              />
            </div>
          );
        })}
      </CardContent>
    </Card>
  );
}
