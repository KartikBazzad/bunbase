import { useState, useEffect } from "react";
import Editor from "@monaco-editor/react";
import { api } from "../../lib/api";
import { Save, RefreshCw, AlertCircle, CheckCircle } from "lucide-react";

interface RulesEditorProps {
  projectId: string;
  collection: string;
}

export function RulesEditor({ projectId, collection }: RulesEditorProps) {
  // Rules are stored as JSON map in backend: { "read": "true", ... }
  // To verify/edit, we can show them as a JSON object, or a custom DSL view?
  // Let's stick to JSON for now, matching Schema Editor.
  // Format:
  // {
  //   "read": "true",
  //   "create": "request.auth != null",
  //   ...
  // }

  const [rulesStr, setRulesStr] = useState("{}");
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  useEffect(() => {
    loadRules();
  }, [projectId, collection]);

  const loadRules = async () => {
    setLoading(true);
    setError("");
    setSuccess("");
    try {
      const res: any = await api.getCollection(projectId, collection);
      // Response: { name, schema, rules }
      if (res.rules) {
        setRulesStr(JSON.stringify(res.rules, null, 2));
      } else {
        // Default template
        setRulesStr(
          JSON.stringify(
            {
              read: "true",
              list: "true",
              create: "request.auth != null",
              update: "resource.data.owner == request.auth.uid",
              delete: "resource.data.owner == request.auth.uid",
            },
            null,
            2,
          ),
        );
      }
    } catch (err) {
      setError("Failed to load rules");
      setRulesStr("{}");
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      let parsed;
      try {
        parsed = JSON.parse(rulesStr);
      } catch (e) {
        throw new Error(
          "Invalid JSON: Rules must be a JSON object mapping operations to CEL strings.",
        );
      }

      await api.updateCollectionRules(projectId, collection, parsed);
      setSuccess("Rules updated successfully!");
      // Hide success message after 3s
      setTimeout(() => setSuccess(""), 3000);
    } catch (err: any) {
      setError("Failed to save: " + err.message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="flex flex-col h-full bg-base-100 p-6 min-h-0 overflow-hidden">
      <div className="flex items-center justify-between mb-4 flex-none">
        <div>
          <h3 className="font-bold text-lg flex items-center gap-2">
            Security Rules{" "}
            <span className="badge badge-neutral text-xs">CEL</span>
          </h3>
          <p className="text-sm text-base-content/60">
            Define access policies using Common Expression Language (CEL).
          </p>
        </div>
        <div className="flex gap-2">
          <button
            className="btn btn-ghost btn-sm btn-square"
            onClick={loadRules}
            title="Reload Rules"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? "animate-spin" : ""}`} />
          </button>
          <button
            className="btn btn-primary btn-sm"
            onClick={handleSave}
            disabled={saving || loading}
          >
            <Save className="w-4 h-4 mr-1" />
            {saving ? "Saving..." : "Save Rules"}
          </button>
        </div>
      </div>

      {error && (
        <div className="alert alert-error mb-4 flex-none py-2 text-sm">
          <AlertCircle className="w-4 h-4" />
          <span>{error}</span>
        </div>
      )}

      {success && (
        <div className="alert alert-success mb-4 flex-none py-2 text-sm">
          <CheckCircle className="w-4 h-4" />
          <span>{success}</span>
        </div>
      )}

      <div className="flex-1 min-h-0 border border-base-300 rounded-md overflow-hidden relative shadow-sm">
        <Editor
          height="100%"
          defaultLanguage="json"
          value={rulesStr}
          onChange={(value) => setRulesStr(value || "")}
          theme="vs-light" // Dynamic theme support later
          options={{
            minimap: { enabled: false },
            fontSize: 13,
            scrollBeyondLastLine: false,
            automaticLayout: true,
          }}
        />
      </div>

      <div className="mt-2 text-xs text-base-content/50">
        <p>
          Available variables: <code>request.auth</code> (uid, email),{" "}
          <code>resource.data</code> (existing doc), <code>request.time</code>.
        </p>
        <p>
          Operations: <code>read</code>, <code>list</code>, <code>create</code>,{" "}
          <code>update</code>, <code>delete</code>.
        </p>
      </div>
    </div>
  );
}
