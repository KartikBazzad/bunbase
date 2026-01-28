import { Link } from 'react-router-dom';

interface Project {
  id: string;
  name: string;
  slug: string;
  owner_id: string;
  created_at: string;
  updated_at: string;
}

interface ProjectCardProps {
  project: Project;
}

export function ProjectCard({ project }: ProjectCardProps) {
  return (
    <Link to={`/projects/${project.id}`}>
      <div className="card hover:shadow-medium transition-shadow cursor-pointer h-full">
        <div className="card-body">
          <div className="flex items-start justify-between mb-2">
            <h3 className="text-lg font-semibold">{project.name}</h3>
            <span className="badge-primary">Active</span>
          </div>
          <p className="text-sm text-gray-600 mb-4">
            Slug: <code className="text-xs bg-gray-100 px-1.5 py-0.5 rounded">{project.slug}</code>
          </p>
          <p className="text-xs text-gray-500">
            Created {new Date(project.created_at).toLocaleDateString()}
          </p>
        </div>
      </div>
    </Link>
  );
}
