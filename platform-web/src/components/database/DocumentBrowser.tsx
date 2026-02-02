import { useState, useEffect } from "react";
import { api } from "../../lib/api";

interface DocumentBrowserProps {
  projectId: string;
  collection: string;
}

export function DocumentBrowser({
  projectId,
  collection,
}: DocumentBrowserProps) {
  const [documents, setDocuments] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [selectedDoc, setSelectedDoc] = useState<any | null>(null);
  const [isEditing, setIsEditing] = useState(false);
  const [editContent, setEditContent] = useState("");

  useEffect(() => {
    if (collection) {
      loadDocuments();
    } else {
      setDocuments([]);
    }
  }, [projectId, collection]);

  const loadDocuments = async () => {
    try {
      setLoading(true);
      setError("");
      const data = await api.listDocuments(projectId, collection);
      // Assuming Bundoc 'list' returns array of docs
      setDocuments(Array.isArray(data) ? data : (data as any).items || []);
    } catch (err) {
      setError("Failed to load documents");
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = () => {
    setSelectedDoc(null);
    setEditContent("{\n  \n}");
    setIsEditing(true);
  };

  const handleEdit = (doc: any) => {
    setSelectedDoc(doc);
    setEditContent(JSON.stringify(doc, null, 2));
    setIsEditing(true);
  };

  const handleSave = async () => {
    try {
      const data = JSON.parse(editContent);
      if (selectedDoc) {
        // Update
        // Assuming doc has '_id' or 'id'
        const id = selectedDoc._id || selectedDoc.id;
        if (!id) throw new Error("Document has no ID");
        await api.updateDocument(projectId, collection, id, data);
      } else {
        // Create
        await api.createDocument(projectId, collection, data);
      }
      setIsEditing(false);
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

  if (!collection) {
    return (
      <div className="flex items-center justify-center h-full text-gray-500">
        Select a collection to view documents
      </div>
    );
  }

  return (
    <div className="card h-full flex flex-col">
      <div className="card-header flex justify-between items-center">
        <h3 className="font-semibold">{collection}</h3>
        <button onClick={handleCreate} className="btn-sm btn-primary">
          + Add Document
        </button>
      </div>

      <div className="card-body overflow-y-auto flex-1 p-0">
        {loading ? (
          <div className="p-4">
            <div className="spinner"></div>
          </div>
        ) : error ? (
          <div className="p-4 text-error-600">{error}</div>
        ) : documents.length === 0 ? (
          <div className="p-8 text-center text-gray-500">
            No documents in this collection
          </div>
        ) : (
          <table className="w-full text-left text-sm">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="px-4 py-2 font-medium text-gray-500">ID</th>
                <th className="px-4 py-2 font-medium text-gray-500">Preview</th>
                <th className="px-4 py-2 font-medium text-gray-500 text-right">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {documents.map((doc, i) => {
                const id = doc._id || doc.id || `doc-${i}`;
                return (
                  <tr
                    key={id}
                    className="hover:bg-gray-50 cursor-pointer"
                    onClick={() => handleEdit(doc)}
                  >
                    <td className="px-4 py-2 font-mono text-xs text-secondary-600 truncate max-w-[150px]">
                      {id}
                    </td>
                    <td className="px-4 py-2 truncate max-w-[300px] text-gray-600">
                      {JSON.stringify(doc).slice(0, 100)}...
                    </td>
                    <td className="px-4 py-2 text-right">
                      <button
                        onClick={(e) => handleDelete(doc, e)}
                        className="text-gray-400 hover:text-error-600 px-2"
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>

      {isEditing && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-2xl h-[80vh] flex flex-col">
            <div className="px-6 py-4 border-b border-gray-200 flex justify-between items-center">
              <h3 className="font-bold text-lg">
                {selectedDoc ? "Edit Document" : "New Document"}
              </h3>
              <button
                onClick={() => setIsEditing(false)}
                className="text-gray-500 hover:text-gray-700"
              >
                ×
              </button>
            </div>
            <div className="flex-1 p-0 relative">
              <textarea
                className="w-full h-full p-4 font-mono text-sm resize-none focus:outline-none"
                value={editContent}
                onChange={(e) => setEditContent(e.target.value)}
              />
            </div>
            <div className="px-6 py-4 border-t border-gray-200 flex justify-end gap-3">
              <button onClick={() => setIsEditing(false)} className="btn-ghost">
                Cancel
              </button>
              <button onClick={handleSave} className="btn-primary">
                Save Changes
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
