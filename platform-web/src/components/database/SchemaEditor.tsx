import { useState, useEffect } from "react";
import { api } from "../../lib/api";
import { Save, RefreshCw } from "lucide-react";

interface SchemaEditorProps {
  projectId: string;
  collection: string;
}

export function SchemaEditor({ projectId, collection }: SchemaEditorProps) {
  const [schemaStr, setSchemaStr] = useState("");
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    loadSchema();
  }, [projectId, collection]);

  const loadSchema = async () => {
    setLoading(true);
    setError("");
    try {
      const res: any = await api.getCollection(projectId, collection);
      // Response: { name, schema }
      if (res.schema) {
        setSchemaStr(JSON.stringify(res.schema, null, 2));
      } else {
        setSchemaStr("{}");
      }
    } catch (err) {
      setError("Failed to load schema");
      setSchemaStr("{}");
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    setError("");
    try {
      let parsed;
      try {
        parsed = JSON.parse(schemaStr);
      } catch (e) {
        throw new Error("Invalid JSON");
      }

      await api.updateCollectionSchema(projectId, collection, parsed);
      alert("Schema updated!");
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
          <h3 className="font-bold text-lg">Schema Validation</h3>
          <p className="text-sm text-base-content/60">
            Enforce document structure using JSON Schema.
          </p>
        </div>
        <div className="flex gap-2">
          <button
            className="btn btn-ghost btn-sm btn-square"
            onClick={loadSchema}
          >
            <RefreshCw className={`w-4 h-4 ${loading ? "animate-spin" : ""}`} />
          </button>
          <button
            className="btn btn-primary btn-sm"
            onClick={handleSave}
            disabled={saving || loading}
          >
            <Save className="w-4 h-4 mr-1" />
            Save Schema
          </button>
        </div>
      </div>

      {error && <div className="alert alert-error mb-4 flex-none">{error}</div>}

      <div className="flex-1 min-h-0 border border-base-300 rounded-md overflow-hidden relative">
        <textarea
          className="w-full h-full p-4 font-mono text-sm resize-none focus:outline-none bg-base-100 text-base-content"
          value={schemaStr}
          onChange={(e) => setSchemaStr(e.target.value)}
          spellCheck={false}
        />
      </div>
    </div>
  );
}
