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
import { useApplicationKeys } from "../../hooks/useApplicationKeys";
import { maskApiKey } from "../../lib/api-keys";
import { Copy, Download, Key, AlertCircle } from "lucide-react";
import { toast } from "sonner";
import { Alert, AlertDescription } from "../ui/alert";
import { Separator } from "../ui/separator";

interface ApplicationConfigDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  applicationId: string;
  applicationName: string;
}

export function ApplicationConfigDialog({
  open,
  onOpenChange,
  applicationId,
  applicationName,
}: ApplicationConfigDialogProps) {
  const { key, isLoading, generateKey, revokeKey, isGenerating, isRevoking } =
    useApplicationKeys(applicationId);

  const [generatedKey, setGeneratedKey] = useState<string | null>(null);
  const [showKey, setShowKey] = useState(false);

  // Reset generated key when dialog closes
  useEffect(() => {
    if (!open) {
      setGeneratedKey(null);
      setShowKey(false);
    }
  }, [open]);

  const handleGenerate = async () => {
    try {
      const result = await generateKey();
      if (result?.data) {
        setGeneratedKey(result.data.key);
        setShowKey(true);
      }
    } catch (error) {
      // Error is handled by the hook
    }
  };

  const handleCopy = (text: string) => {
    navigator.clipboard.writeText(text);
    toast.success("Copied to clipboard");
  };

  const handleDownloadEnv = () => {
    if (!generatedKey) return;

    const content = `BUNBASE_API_KEY=${generatedKey}\nBUNBASE_APPLICATION_ID=${applicationId}\n`;
    const blob = new Blob([content], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `.env`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    toast.success("Downloaded .env file");
  };

  const handleDownloadJson = () => {
    if (!generatedKey) return;

    const content = JSON.stringify(
      {
        apiKey: generatedKey,
        applicationId: applicationId,
        applicationName: applicationName,
      },
      null,
      2,
    );
    const blob = new Blob([content], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `bunbase-config.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    toast.success("Downloaded config.json file");
  };

  const handleRevoke = async () => {
    try {
      await revokeKey();
      setGeneratedKey(null);
      setShowKey(false);
    } catch (error) {
      // Error is handled by the hook
    }
  };

  const maskedKey = key ? maskApiKey(key.keyPrefix, key.keySuffix) : null;

  const displayKey = generatedKey || (key ? maskedKey : null);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Application Configuration</DialogTitle>
          <DialogDescription>
            Manage API keys and configuration for {applicationName}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6 py-4">
          {/* Current Key Status */}
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
            </div>
          ) : key && !generatedKey ? (
            <div className="space-y-4">
              <Alert>
                <Key className="h-4 w-4" />
                <AlertDescription>
                  An API key exists for this application. The full key cannot be
                  retrieved after generation.
                </AlertDescription>
              </Alert>
              <div className="space-y-2">
                <Label>Current API Key</Label>
                <div className="flex gap-2">
                  <Input
                    value={maskedKey || ""}
                    readOnly
                    className="font-mono text-sm"
                  />
                  <Button
                    variant="outline"
                    size="icon"
                    onClick={() => handleCopy(maskedKey || "")}
                    title="Copy masked key"
                  >
                    <Copy className="h-4 w-4" />
                  </Button>
                </div>
                <p className="text-xs text-muted-foreground">
                  Created: {new Date(key.createdAt).toLocaleDateString()}
                </p>
              </div>
            </div>
          ) : generatedKey ? (
            <div className="space-y-4">
              <Alert className="border-yellow-500 bg-yellow-50 dark:bg-yellow-950">
                <AlertCircle className="h-4 w-4 text-yellow-600 dark:text-yellow-400" />
                <AlertDescription className="text-yellow-800 dark:text-yellow-200">
                  <strong>Important:</strong> This is the only time you can see
                  this API key. Make sure to copy it now or download the
                  configuration file.
                </AlertDescription>
              </Alert>
              <div className="space-y-2">
                <Label>Your API Key</Label>
                <div className="flex gap-2">
                  <Input
                    value={generatedKey}
                    readOnly
                    className="font-mono text-sm"
                  />
                  <Button
                    variant="outline"
                    size="icon"
                    onClick={() => handleCopy(generatedKey)}
                  >
                    <Copy className="h-4 w-4" />
                  </Button>
                </div>
              </div>
              <Separator />
              <div className="space-y-3">
                <Label>Download Configuration</Label>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    onClick={handleDownloadEnv}
                    className="flex-1"
                  >
                    <Download className="mr-2 h-4 w-4" />
                    Download .env
                  </Button>
                  <Button
                    variant="outline"
                    onClick={handleDownloadJson}
                    className="flex-1"
                  >
                    <Download className="mr-2 h-4 w-4" />
                    Download JSON
                  </Button>
                </div>
              </div>
              <Separator />
              <div className="space-y-3">
                <Label>SDK Initialization</Label>
                <div className="space-y-2">
                  <div>
                    <p className="text-xs text-muted-foreground mb-1">
                      JavaScript / TypeScript
                    </p>
                    <div className="relative">
                      <pre className="bg-muted p-3 rounded-md text-xs overflow-x-auto">
                        <code>{`import { BunBase } from '@bunbase/js-sdk';

const client = new BunBase({
  apiKey: '${generatedKey}',
  applicationId: '${applicationId}'
});`}</code>
                      </pre>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="absolute top-2 right-2 h-6 w-6"
                        onClick={() =>
                          handleCopy(
                            `import { BunBase } from '@bunbase/js-sdk';\n\nconst client = new BunBase({\n  apiKey: '${generatedKey}',\n  applicationId: '${applicationId}'\n});`,
                          )
                        }
                      >
                        <Copy className="h-3 w-3" />
                      </Button>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          ) : (
            <Alert>
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>
                No API key has been generated for this application yet.
              </AlertDescription>
            </Alert>
          )}

          {/* Actions */}
          <div className="flex gap-2 pt-4">
            {key && !generatedKey && (
              <Button
                variant="destructive"
                onClick={handleRevoke}
                disabled={isRevoking}
                className="flex-1"
              >
                {isRevoking ? "Revoking..." : "Revoke Key"}
              </Button>
            )}
            <Button
              onClick={handleGenerate}
              disabled={isGenerating || (generatedKey !== null && !showKey)}
              className="flex-1"
            >
              {generatedKey && showKey
                ? "Key Generated"
                : isGenerating
                  ? "Generating..."
                  : key
                    ? "Regenerate Key"
                    : "Generate API Key"}
            </Button>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
