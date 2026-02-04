import { useState, useCallback } from "react";
import { Braces, Plus, Trash2 } from "lucide-react";

const MAX_DEPTH = 4;

type JsonType = "string" | "number" | "boolean" | "null" | "array" | "object";

function getValueType(v: unknown): JsonType {
  if (v === null) return "null";
  if (Array.isArray(v)) return "array";
  if (typeof v === "object") return "object";
  if (typeof v === "string") return "string";
  if (typeof v === "number") return "number";
  if (typeof v === "boolean") return "boolean";
  return "null";
}

function defaultForType(t: Exclude<JsonType, "null">): unknown {
  switch (t) {
    case "string":
      return "";
    case "number":
      return 0;
    case "boolean":
      return false;
    case "array":
      return [];
    case "object":
      return {};
    default:
      return null;
  }
}

export interface DocumentEditorProps {
  value: Record<string, unknown>;
  onChange: (doc: Record<string, unknown>) => void;
  readOnlyId?: boolean;
  depth?: number;
}

export function DocumentEditor({
  value,
  onChange,
  readOnlyId = false,
  depth = 0,
}: DocumentEditorProps) {
  const [jsonMode, setJsonMode] = useState(false);
  const [jsonText, setJsonText] = useState("");
  const [jsonError, setJsonError] = useState("");
  const [addKey, setAddKey] = useState("");
  const [addType, setAddType] = useState<Exclude<JsonType, "null">>("string");
  const [showAdd, setShowAdd] = useState(false);
  const [addError, setAddError] = useState("");

  const updateKey = useCallback(
    (key: string, newVal: unknown) => {
      const next = { ...value, [key]: newVal };
      onChange(next);
    },
    [value, onChange]
  );

  const removeKey = useCallback(
    (key: string) => {
      const next = { ...value };
      delete next[key];
      onChange(next);
    },
    [value, onChange]
  );

  const addField = useCallback(() => {
    const k = addKey.trim();
    if (!k) return;
    if (Object.prototype.hasOwnProperty.call(value, k)) {
      setAddError("Key already exists");
      return;
    }
    setAddError("");
    const next = { ...value, [k]: defaultForType(addType) };
    onChange(next);
    setAddKey("");
    setAddType("string");
    setShowAdd(false);
  }, [addKey, addType, value, onChange]);

  const enterJsonMode = useCallback(() => {
    setJsonText(JSON.stringify(value, null, 2));
    setJsonError("");
    setJsonMode(true);
  }, [value]);

  const applyJson = useCallback(() => {
    try {
      const parsed = JSON.parse(jsonText) as Record<string, unknown>;
      if (
        typeof parsed !== "object" ||
        parsed === null ||
        Array.isArray(parsed)
      ) {
        setJsonError("Root must be a JSON object");
        return;
      }
      setJsonError("");
      onChange(parsed);
    } catch (e) {
      setJsonError(e instanceof Error ? e.message : "Invalid JSON");
    }
  }, [jsonText, onChange]);

  const exitJsonMode = useCallback(() => {
    setJsonMode(false);
    setJsonError("");
  }, []);

  const keys = Object.keys(value);

  if (jsonMode) {
    return (
      <div className="flex flex-col gap-2 h-full min-h-0">
        <div className="flex justify-between items-center flex-none">
          <button
            type="button"
            className="btn btn-ghost btn-sm gap-1"
            onClick={() => {
              applyJson();
              exitJsonMode();
            }}
          >
            Apply & switch to Form
          </button>
          <button
            type="button"
            className="btn btn-ghost btn-sm"
            onClick={exitJsonMode}
          >
            Cancel (discard JSON changes)
          </button>
        </div>
        <textarea
          className="textarea textarea-bordered font-mono text-sm flex-1 min-h-[200px] resize-none"
          value={jsonText}
          onChange={(e) => setJsonText(e.target.value)}
          spellCheck={false}
        />
        {jsonError && <p className="text-error text-sm">{jsonError}</p>}
        <button
          type="button"
          className="btn btn-primary btn-sm self-start"
          onClick={applyJson}
        >
          Apply
        </button>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3 h-full min-h-0 overflow-auto">
      <div className="flex justify-end flex-none">
        <button
          type="button"
          className="btn btn-ghost btn-sm gap-1"
          onClick={enterJsonMode}
          aria-label="Edit as JSON"
        >
          <Braces className="w-4 h-4" />
          Edit as JSON
        </button>
      </div>
      <div className="space-y-3 flex-1 min-h-0">
        {keys.map((key) => (
          <FieldRow
            key={key}
            fieldKey={key}
            fieldValue={value[key]}
            readOnlyValue={key === "_id" && readOnlyId}
            onValueChange={(v) => updateKey(key, v)}
            onRemove={
              key === "_id" && readOnlyId ? undefined : () => removeKey(key)
            }
            depth={depth}
          />
        ))}
        {showAdd ? (
          <div className="flex flex-wrap items-center gap-2 p-2 bg-base-200 rounded-lg">
            <input
              type="text"
              placeholder="Field name"
              value={addKey}
              onChange={(e) => setAddKey(e.target.value)}
              className="input input-bordered input-sm w-32"
              autoFocus
            />
            <select
              value={addType}
              onChange={(e) =>
                setAddType(e.target.value as Exclude<JsonType, "null">)
              }
              className="select select-bordered select-sm w-28"
            >
              <option value="string">string</option>
              <option value="number">number</option>
              <option value="boolean">boolean</option>
              <option value="array">array</option>
              <option value="object">object</option>
            </select>
            <button
              type="button"
              className="btn btn-primary btn-sm"
              onClick={addField}
            >
              Add
            </button>
            <button
              type="button"
              className="btn btn-ghost btn-sm"
              onClick={() => {
                setShowAdd(false);
                setAddKey("");
                setAddError("");
              }}
            >
              Cancel
            </button>
            {addError && <span className="text-error text-xs">{addError}</span>}
          </div>
        ) : (
          <button
            type="button"
            className="btn btn-ghost btn-sm gap-1 text-base-content/70"
            onClick={() => setShowAdd(true)}
          >
            <Plus className="w-4 h-4" />
            Add field
          </button>
        )}
      </div>
    </div>
  );
}

interface FieldRowProps {
  fieldKey: string;
  fieldValue: unknown;
  readOnlyValue: boolean;
  onValueChange: (v: unknown) => void;
  onRemove?: () => void;
  depth: number;
}

function FieldRow({
  fieldKey,
  fieldValue,
  readOnlyValue,
  onValueChange,
  onRemove,
  depth,
}: FieldRowProps) {
  const type = getValueType(fieldValue);

  return (
    <div className="flex flex-col gap-1 border border-base-300 rounded-lg p-2">
      <div className="flex items-center justify-between gap-2">
        <span className="font-mono text-sm font-medium text-base-content/80 shrink-0">
          {fieldKey}
        </span>
        {onRemove && (
          <button
            type="button"
            className="btn btn-ghost btn-xs text-error shrink-0"
            onClick={onRemove}
            aria-label={`Remove ${fieldKey}`}
          >
            <Trash2 className="w-3 h-3" />
          </button>
        )}
      </div>
      <div className="min-w-0">
        <ValueEditor
          type={type}
          value={fieldValue}
          onChange={onValueChange}
          depth={depth}
          readOnly={readOnlyValue}
        />
      </div>
    </div>
  );
}

interface ValueEditorProps {
  type: JsonType;
  value: unknown;
  onChange: (v: unknown) => void;
  depth: number;
  readOnly?: boolean;
}

function ValueEditor({
  type,
  value,
  onChange,
  depth,
  readOnly = false,
}: ValueEditorProps) {
  if (type === "string") {
    const s = typeof value === "string" ? value : "";
    const isLong = s.length > 80;
    return (
      <input
        type="text"
        className={`input input-bordered input-sm w-full font-mono ${isLong ? "min-h-16" : ""}`}
        value={s}
        onChange={(e) => onChange(e.target.value)}
        readOnly={readOnly}
        disabled={readOnly}
      />
    );
  }
  if (type === "number") {
    const n = typeof value === "number" ? value : 0;
    return (
      <input
        type="number"
        step="any"
        className="input input-bordered input-sm w-full font-mono"
        value={n}
        onChange={(e) => {
          const v = e.target.value;
          if (v === "" || v === "-") return;
          const num = Number(v);
          if (!Number.isNaN(num)) onChange(num);
        }}
        readOnly={readOnly}
        disabled={readOnly}
      />
    );
  }
  if (type === "boolean") {
    const b = value === true;
    return (
      <label className="flex items-center gap-2 cursor-pointer">
        <input
          type="checkbox"
          className="checkbox checkbox-sm"
          checked={b}
          onChange={(e) => onChange(e.target.checked)}
          disabled={readOnly}
        />
        <span className="text-sm">{b ? "true" : "false"}</span>
      </label>
    );
  }
  if (type === "null") {
    return (
      <div className="flex items-center gap-2 flex-wrap">
        <span className="text-base-content/50 text-sm">null</span>
        <select
          className="select select-bordered select-xs w-36"
          value=""
          onChange={(e) => {
            const t = e.target.value as Exclude<JsonType, "null">;
            if (t) onChange(defaultForType(t));
          }}
        >
          <option value="">Change type…</option>
          <option value="string">string</option>
          <option value="number">number</option>
          <option value="boolean">boolean</option>
          <option value="array">array</option>
          <option value="object">object</option>
        </select>
      </div>
    );
  }
  if (type === "array") {
    const arr = Array.isArray(value) ? value : [];
    return (
      <ArrayEditor
        value={arr}
        onChange={onChange}
        depth={depth}
        readOnly={readOnly}
      />
    );
  }
  if (type === "object") {
    const obj =
      value && typeof value === "object" && !Array.isArray(value)
        ? (value as Record<string, unknown>)
        : {};
    return (
      <ObjectEditor
        value={obj}
        onChange={onChange}
        depth={depth}
        readOnly={readOnly}
      />
    );
  }
  return null;
}

interface ArrayEditorProps {
  value: unknown[];
  onChange: (v: unknown) => void;
  depth: number;
  readOnly?: boolean;
}

function ArrayEditor({
  value,
  onChange,
  depth,
  readOnly = false,
}: ArrayEditorProps) {
  const [editingIndex, setEditingIndex] = useState<number | null>(null);

  const updateItem = (index: number, item: unknown) => {
    const next = [...value];
    next[index] = item;
    onChange(next);
    setEditingIndex(null);
  };

  const removeItem = (index: number) => {
    const next = value.filter((_, i) => i !== index);
    onChange(next);
    setEditingIndex(null);
  };

  const addItem = (t: Exclude<JsonType, "null">) => {
    onChange([...value, defaultForType(t)]);
  };

  if (depth >= MAX_DEPTH) {
    return (
      <div className="text-sm text-base-content/60">
        Nested too deep — use “Edit as JSON” for this array.
      </div>
    );
  }

  return (
    <div className="space-y-2 pl-2 border-l-2 border-base-300">
      {value.map((item, i) => (
        <div key={i} className="flex items-start gap-2">
          <span className="text-base-content/50 text-xs shrink-0 w-6">{i}</span>
          {editingIndex === i ? (
            <div className="flex-1 min-w-0">
              <ValueEditor
                type={getValueType(item)}
                value={item}
                onChange={(v) => updateItem(i, v)}
                depth={depth + 1}
              />
              {!readOnly && (
                <button
                  type="button"
                  className="btn btn-ghost btn-xs mt-1"
                  onClick={() => setEditingIndex(null)}
                >
                  Done
                </button>
              )}
            </div>
          ) : (
            <>
              <div className="flex-1 min-w-0 truncate text-sm font-mono text-base-content/80">
                {getValueType(item) === "object" ||
                getValueType(item) === "array"
                  ? JSON.stringify(item).slice(0, 60) +
                    (JSON.stringify(item).length > 60 ? "…" : "")
                  : String(item)}
              </div>
              {!readOnly && (
                <>
                  <button
                    type="button"
                    className="btn btn-ghost btn-xs shrink-0"
                    onClick={() => setEditingIndex(i)}
                  >
                    Edit
                  </button>
                  <button
                    type="button"
                    className="btn btn-ghost btn-xs text-error shrink-0"
                    onClick={() => removeItem(i)}
                    aria-label={`Remove item ${i}`}
                  >
                    <Trash2 className="w-3 h-3" />
                  </button>
                </>
              )}
            </>
          )}
        </div>
      ))}
      {!readOnly && (
        <div className="flex items-center gap-2">
          <span className="text-base-content/50 text-xs w-6 shrink-0" />
          <select
            className="select select-bordered select-xs w-28"
            value=""
            onChange={(e) => {
              const t = e.target.value as Exclude<JsonType, "null">;
              if (t) addItem(t);
              e.target.value = "";
            }}
          >
            <option value="">Add item…</option>
            <option value="string">string</option>
            <option value="number">number</option>
            <option value="boolean">boolean</option>
            <option value="array">array</option>
            <option value="object">object</option>
          </select>
        </div>
      )}
    </div>
  );
}

interface ObjectEditorProps {
  value: Record<string, unknown>;
  onChange: (v: unknown) => void;
  depth: number;
  readOnly?: boolean;
}

function ObjectEditor({ value, onChange, depth }: ObjectEditorProps) {
  if (depth >= MAX_DEPTH) {
    return (
      <div className="text-sm text-base-content/60">
        Nested too deep — use “Edit as JSON” for this object.
      </div>
    );
  }

  return (
    <div className="pl-2 border-l-2 border-base-300">
      <DocumentEditor
        value={value}
        onChange={(doc) => onChange(doc)}
        readOnlyId={false}
        depth={depth + 1}
      />
    </div>
  );
}
