import { useState, useEffect, useMemo } from "react";
import { api } from "../../lib/api";
import {
  FilePlus,
  Trash2,
  Pencil,
  Search,
  ChevronRight,
  Database,
  List,
  Shield,
  FileJson,
} from "lucide-react";
import { DocumentEditor } from "./DocumentEditor";
import { IndexesTab } from "./IndexesTab";
import { SchemaEditor } from "./SchemaEditor";
import { RulesEditor } from "./RulesEditor";

const PAGE_SIZE = 20;
const MAX_DATA_COLUMNS = 5;

function getDataColumns(documents: any[]): string[] {
  if (documents.length === 0) return [];
  const doc = documents[0];
  const keys = Object.keys(doc).filter(
    (k) => k !== "_id" && k !== "id",
  ) as string[];
  return keys.slice(0, MAX_DATA_COLUMNS);
}

function isBadgeKey(key: string): boolean {
  const lower = key.toLowerCase();
  return (
    lower === "role" ||
    lower === "type" ||
    lower === "status" ||
    lower.endsWith("_role") ||
    lower.endsWith("_type") ||
    lower.endsWith("_status")
  );
}

function formatCellValue(value: unknown): React.ReactNode {
  if (value === null || value === undefined) return "—";
  if (typeof value === "object") return JSON.stringify(value);
  return String(value);
}

interface DocumentBrowserProps {
  projectId: string;
  collection: string;
  projectName?: string;
  onDocumentsLoaded?: (count: number) => void;
}

export function DocumentBrowser({
  projectId,
  collection,
  projectName = "Project",
  onDocumentsLoaded,
}: DocumentBrowserProps) {
  const [documents, setDocuments] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [selectedDoc, setSelectedDoc] = useState<any | null>(null);
  const [isEditing, setIsEditing] = useState(false);
  const [documentObject, setDocumentObject] = useState<Record<
    string,
    unknown
  > | null>(null);
  const [activeTab, setActiveTab] = useState<
    "data" | "rules" | "indexes" | "schema"
  >("data");
  const [currentPage, setCurrentPage] = useState(1);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [searchQuery, setSearchQuery] = useState("");
  const [hasMore, setHasMore] = useState(true);

  useEffect(() => {
    // Reset page when collection changes
    setCurrentPage(1);
    setSearchQuery("");
  }, [projectId, collection]);

  useEffect(() => {
    if (collection && activeTab === "data") {
      loadDocuments();
    } else {
      // If switching tabs, maybe do nothing or clear docs?
      // Keep docs in state for quick switch back
    }
  }, [projectId, collection, activeTab, currentPage]); // Add searchQuery debounce?

  // Search trigger
  const handlesearch = () => {
    setCurrentPage(1);
    loadDocuments();
  };

  const loadDocuments = async () => {
    if (!collection) return;
    try {
      setLoading(true);
      setError("");

      const skip = (currentPage - 1) * PAGE_SIZE;
      const limit = PAGE_SIZE;

      let list = [];

      if (searchQuery.trim()) {
        // Try to parse query as JSON
        let queryObj = {};
        try {
          queryObj = JSON.parse(searchQuery);
          // If valid JSON, use it
        } catch (e) {
          // If not JSON, maybe basic field search?
          // For now, simple text filter needs structure.
          // Let's assume user types JSON like {"email": "alice"}
          // If invalid, we can error or search nothing.
          // setError("Invalid JSON query"); return;
          // Fallback: don't query?
        }

        if (Object.keys(queryObj).length > 0) {
          const res: any = await api.queryDocuments(
            projectId,
            collection,
            queryObj,
            { skip, limit },
          );
          list = res.documents || res || []; // Adapt to response
        } else {
          // Empty query -> List
          const data = await api.listDocuments(projectId, collection, {
            skip,
            limit,
          });
          list = Array.isArray(data) ? data : (data as any).documents || [];
        }
      } else {
        const data = await api.listDocuments(projectId, collection, {
          skip,
          limit,
        });
        list = Array.isArray(data) ? data : (data as any).documents || [];
      }

      setDocuments(list);
      onDocumentsLoaded?.(list.length); // Total count unknown without count API

      // Heuristic for hasMore
      setHasMore(list.length === PAGE_SIZE);
    } catch (err) {
      setError("Failed to load documents");
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = () => {
    setSelectedDoc(null);
    setDocumentObject({});
    setIsEditing(true);
  };

  const handleEdit = (doc: any) => {
    setSelectedDoc(doc);
    setDocumentObject(
      typeof doc === "object" && doc !== null
        ? (JSON.parse(JSON.stringify(doc)) as Record<string, unknown>)
        : {},
    );
    setIsEditing(true);
  };

  const handleSave = async () => {
    const data = documentObject ?? {};
    try {
      if (selectedDoc) {
        const id = selectedDoc._id || selectedDoc.id;
        if (!id) throw new Error("Document has no ID");
        await api.updateDocument(projectId, collection, id, data);
      } else {
        await api.createDocument(projectId, collection, data);
      }
      setIsEditing(false);
      setDocumentObject(null);
      loadDocuments();
    } catch (err) {
      alert(
        "Failed to save: " + (err instanceof Error ? err.message : String(err)),
      );
    }
  };

  const handleDelete = async (doc: any, e: React.MouseEvent) => {
    e.stopPropagation();
    const id = doc._id || doc.id;
    if (!confirm(`Delete document ${id}?`)) return;
    try {
      await api.deleteDocument(projectId, collection, id);
      loadDocuments();
    } catch (err) {
      alert("Failed to delete");
    }
  };

  const dataColumns = useMemo(() => getDataColumns(documents), [documents]);
  // Pagination is server-side now. "totalPages" is unknown. We use Prev/Next.

  const toggleSelectAll = () => {
    if (selectedIds.size === documents.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(
        new Set(documents.map((d) => String(d._id ?? d.id ?? ""))),
      );
    }
  };

  const toggleSelect = (id: string) => {
    const next = new Set(selectedIds);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    setSelectedIds(next);
  };

  if (!collection) {
    return (
      <div className="flex items-center justify-center h-full min-h-[200px] text-base-content/50">
        Select a collection to view documents
      </div>
    );
  }

  return (
    <div className="bg-base-100 rounded-lg border border-base-300 h-full flex flex-col min-h-0 overflow-hidden">
      {/* Breadcrumbs */}
      <div className="flex items-center gap-2 text-sm text-base-content/70 px-4 pt-4 pb-1 flex-none">
        <span>{projectName}</span>
        <ChevronRight className="w-4 h-4 opacity-50" />
        <span>Database</span>
        <ChevronRight className="w-4 h-4 opacity-50" />
        <span className="text-base-content font-medium">{collection}</span>
      </div>

      {/* Title + subtitle */}
      <div className="px-4 pb-4 flex-none">
        <h1 className="text-2xl font-bold text-base-content">{collection}</h1>
        <p className="text-sm text-base-content/60 mt-0.5">
          Manage your documents, schemas, and indexes.
        </p>
      </div>

      {/* Tabs */}
      <div className="tabs tabs-boxed px-4 flex-none bg-base-200/50 rounded-t-lg mb-2 mx-4 w-auto inline-flex">
        <button
          type="button"
          className={`tab tab-sm ${activeTab === "data" ? "tab-active" : ""}`}
          onClick={() => setActiveTab("data")}
        >
          <Database className="w-4 h-4 mr-2" />
          Data
        </button>
        <button
          type="button"
          className={`tab tab-sm ${activeTab === "schema" ? "tab-active" : ""}`}
          onClick={() => setActiveTab("schema")}
        >
          <FileJson className="w-4 h-4 mr-2" />
          Schema
        </button>
        <button
          type="button"
          className={`tab tab-sm ${activeTab === "indexes" ? "tab-active" : ""}`}
          onClick={() => setActiveTab("indexes")}
        >
          <List className="w-4 h-4 mr-2" />
          Indexes
        </button>
        <button
          type="button"
          className={`tab tab-sm ${activeTab === "rules" ? "tab-active" : ""}`}
          onClick={() => setActiveTab("rules")}
        >
          <Shield className="w-4 h-4 mr-2" />
          Rules
        </button>
      </div>

      {/* Tab content */}
      {activeTab === "rules" && (
        <RulesEditor projectId={projectId} collection={collection} />
      )}
      {activeTab === "indexes" && (
        <IndexesTab projectId={projectId} collection={collection} />
      )}
      {activeTab === "schema" && (
        <SchemaEditor projectId={projectId} collection={collection} />
      )}
      {activeTab === "data" && (
        <>
          {/* Action bar */}
          <div className="flex flex-wrap items-center gap-3 px-4 pb-3 flex-none">
            <div className="flex-1 min-w-[200px] relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-base-content/40" />
              <input
                type="text"
                placeholder='Query (e.g. {"status": "active"})'
                value={searchQuery}
                onKeyDown={(e) => e.key === "Enter" && handlesearch()}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="input input-bordered input-sm w-full pl-9 font-mono"
              />
            </div>
            <button
              type="button"
              className="btn btn-primary btn-sm btn-square"
              onClick={handlesearch}
            >
              <ChevronRight className="w-4 h-4" />
            </button>
            <button
              type="button"
              onClick={handleCreate}
              className="btn btn-primary btn-sm gap-1 ml-2"
            >
              <FilePlus className="w-3.5 h-3.5" />
              Add Document
            </button>
          </div>

          <div className="flex-1 min-h-0 overflow-auto border border-base-300 border-t-0 rounded-b-lg">
            {loading ? (
              <div className="p-6 flex justify-center">
                <span className="loading loading-spinner" />
              </div>
            ) : error ? (
              <div className="p-6 text-error">{error}</div>
            ) : documents.length === 0 ? (
              <div className="p-8 text-center text-base-content/50">
                No documents found.
              </div>
            ) : (
              <table className="table table-pin-rows table-xs text-left">
                <thead className="bg-base-200 sticky top-0 z-10">
                  <tr>
                    <th className="w-10">
                      <input
                        type="checkbox"
                        className="checkbox checkbox-sm"
                        checked={
                          documents.length > 0 &&
                          documents.every((d) =>
                            selectedIds.has(String(d._id ?? d.id ?? "")),
                          )
                        }
                        onChange={toggleSelectAll}
                        aria-label="Select all"
                      />
                    </th>
                    <th className="font-semibold text-base-content/70 uppercase text-xs">
                      Document ID
                    </th>
                    {dataColumns.map((col) => (
                      <th
                        key={col}
                        className="font-semibold text-base-content/70 uppercase text-xs"
                      >
                        {col.replace(/_/g, " ")}
                      </th>
                    ))}
                    <th className="text-right font-semibold text-base-content/70 uppercase text-xs w-24">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {documents.map((doc, i) => {
                    const id = String(doc._id ?? doc.id ?? `doc-${i}`);
                    return (
                      <tr
                        key={id}
                        className="hover cursor-pointer"
                        onClick={() => handleEdit(doc)}
                      >
                        <td onClick={(e) => e.stopPropagation()}>
                          <input
                            type="checkbox"
                            className="checkbox checkbox-sm"
                            checked={selectedIds.has(id)}
                            onChange={() => toggleSelect(id)}
                            aria-label={`Select ${id}`}
                          />
                        </td>
                        <td>
                          <button
                            type="button"
                            className="link link-primary font-mono text-xs text-left"
                            onClick={(e) => {
                              e.stopPropagation();
                              handleEdit(doc);
                            }}
                          >
                            {id}
                          </button>
                        </td>
                        {dataColumns.map((col) => {
                          const val = doc[col];
                          const content =
                            isBadgeKey(col) && typeof val === "string" ? (
                              <span className="badge badge-sm badge-ghost">
                                {val}
                              </span>
                            ) : (
                              formatCellValue(val)
                            );
                          return (
                            <td
                              key={col}
                              className="max-w-[180px] truncate text-base-content/80"
                              title={
                                typeof val === "object"
                                  ? JSON.stringify(val)
                                  : String(val)
                              }
                            >
                              {content}
                            </td>
                          );
                        })}
                        <td
                          className="text-right"
                          onClick={(e) => e.stopPropagation()}
                        >
                          <button
                            type="button"
                            className="btn btn-ghost btn-xs text-base-content/60 hover:text-primary"
                            onClick={() => handleEdit(doc)}
                            aria-label="Edit"
                          >
                            <Pencil className="w-3 h-3" />
                          </button>
                          <button
                            type="button"
                            className="btn btn-ghost btn-xs text-base-content/60 hover:text-error"
                            onClick={(e) => handleDelete(doc, e)}
                            aria-label="Delete"
                          >
                            <Trash2 className="w-3 h-3" />
                          </button>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            )}
          </div>

          {/* Footer: pagination */}
          {!loading && !error && (
            <div className="flex-none flex flex-wrap items-center justify-between gap-3 px-4 py-3 border-t border-base-300 bg-base-200/30">
              <div className="text-sm text-base-content/70">
                Page {currentPage}
              </div>
              <div className="flex items-center gap-4">
                <div className="join">
                  <button
                    type="button"
                    className="join-item btn btn-sm btn-ghost"
                    disabled={currentPage <= 1}
                    onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
                  >
                    Previous
                  </button>
                  <button
                    type="button"
                    className="join-item btn btn-sm btn-ghost"
                    disabled={!hasMore}
                    onClick={() => setCurrentPage((p) => p + 1)}
                  >
                    Next
                  </button>
                </div>
              </div>
            </div>
          )}
        </>
      )}

      {isEditing && documentObject !== null && (
        <dialog className="modal modal-open">
          <div className="modal-box w-full max-w-2xl h-[80vh] flex flex-col">
            <h3 className="font-bold text-lg mb-4">
              {selectedDoc ? "Edit Document" : "New Document"}
            </h3>
            <div className="flex-1 min-h-0 overflow-hidden flex flex-col">
              <DocumentEditor
                value={documentObject}
                onChange={setDocumentObject}
                readOnlyId={!!selectedDoc}
              />
            </div>
            <div className="modal-action">
              <button
                type="button"
                onClick={() => {
                  setIsEditing(false);
                  setDocumentObject(null);
                }}
                className="btn btn-ghost"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleSave}
                className="btn btn-primary"
              >
                Save Changes
              </button>
            </div>
          </div>
          <form method="dialog" className="modal-backdrop">
            <button
              type="button"
              onClick={() => {
                setIsEditing(false);
                setDocumentObject(null);
              }}
            >
              close
            </button>
          </form>
        </dialog>
      )}
    </div>
  );
}
