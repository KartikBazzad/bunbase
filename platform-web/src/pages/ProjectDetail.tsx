import { Navigate, useParams } from "react-router-dom";

export function ProjectDetail() {
  const { id } = useParams<{ id: string }>();

  return <Navigate to={`/projects/${id}/overview`} replace />;
}
