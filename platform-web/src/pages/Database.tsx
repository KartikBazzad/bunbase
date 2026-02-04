import { useState, useEffect } from "react";
import { useParams } from "react-router-dom";
import { CollectionList } from "../components/database/CollectionList";
import { DocumentBrowser } from "../components/database/DocumentBrowser";
import { api } from "../lib/api";

export function Database() {
  const { id: projectId } = useParams<{ id: string }>();
  const [selectedCollection, setSelectedCollection] = useState<string>("");
  const [selectedCollectionCount, setSelectedCollectionCount] = useState<
    number | undefined
  >(undefined);
  const [projectName, setProjectName] = useState<string>("Project");

  useEffect(() => {
    if (projectId) {
      api
        .getProject(projectId)
        .then((p: any) => setProjectName(p?.name ?? "Project"))
        .catch(() => setProjectName("Project"));
    }
  }, [projectId]);

  useEffect(() => {
    setSelectedCollectionCount(undefined);
  }, [selectedCollection]);

  const handleDocumentsLoaded = (count: number) => {
    setSelectedCollectionCount(count);
  };

  if (!projectId) {
    return <div className="p-6 text-base-content/70">No project selected.</div>;
  }

  return (
    <div className="flex-1 min-h-0 flex flex-col">
      <div className="flex-1 grid grid-cols-1 lg:grid-cols-[280px_1fr] gap-4 min-h-0">
        <div className="min-h-[200px] lg:min-h-0 lg:h-full overflow-hidden flex flex-col">
          <CollectionList
            projectId={projectId}
            selectedCollection={selectedCollection}
            selectedCollectionCount={selectedCollectionCount}
            onSelectCollection={setSelectedCollection}
          />
        </div>
        <div className="min-h-[300px] lg:min-h-0 lg:h-full overflow-hidden flex flex-col">
          <DocumentBrowser
            projectId={projectId}
            collection={selectedCollection}
            projectName={projectName}
            onDocumentsLoaded={handleDocumentsLoaded}
          />
        </div>
      </div>
    </div>
  );
}
