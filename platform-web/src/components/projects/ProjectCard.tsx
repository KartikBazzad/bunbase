import { Link } from 'react-router-dom';
import { FolderKanban, Calendar } from 'lucide-react';

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
      <div className="card bg-base-100 shadow-md hover:shadow-lg transition-shadow cursor-pointer h-full">
        <div className="card-body">
          <div className="flex items-start justify-between mb-2">
            <div className="flex items-center gap-2">
              <FolderKanban className="w-5 h-5 text-primary" />
              <h3 className="text-lg font-semibold">{project.name}</h3>
            </div>
            <div className="badge badge-primary">Active</div>
          </div>
          <p className="text-sm text-base-content/70 mb-4">
            Slug: <code className="text-xs bg-base-300 px-1.5 py-0.5 rounded">{project.slug}</code>
          </p>
          <div className="flex items-center text-xs text-base-content/50">
            <Calendar className="w-3 h-3 mr-1" />
            <span>Created {new Date(project.created_at).toLocaleDateString()}</span>
          </div>
        </div>
      </div>
    </Link>
  );
}
