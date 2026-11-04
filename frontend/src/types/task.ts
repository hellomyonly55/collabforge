import type { TeamRole } from "./team";
import type { Project } from "./project";
import type { Page } from "./team";
export type TaskStatus = "todo" | "in_progress" | "in_review" | "done";
export type TaskPriority = "low" | "medium" | "high" | "urgent";
export interface Task {
  id: string;
  project_id: string;
  project_name: string;
  team_id: string;
  team_name: string;
  my_role: TeamRole;
  project_archived_at: string | null;
  team_archived_at: string | null;
  title: string;
  description: string;
  status: TaskStatus;
  priority: TaskPriority;
  assignee_id: string | null;
  assignee_nickname: string | null;
  created_by: string;
  creator_nickname: string;
  due_at: string | null;
  is_overdue: boolean;
  position: number;
  version: number;
  can_delete: boolean;
  created_at: string;
  updated_at: string;
}
export interface TaskFields {
  title: string;
  description: string;
  priority: TaskPriority;
  assignee_id: string | null;
  due_at: string | null;
}
export interface TaskFilters {
  keyword: string;
  status: string;
  priority: string;
  assignee_id: string;
  overdue: string;
}
export interface TaskPage extends Page<Task> {
  revision?: number;
}
export interface BoardColumn {
  status: TaskStatus;
  items: Task[];
  total: number;
  page_size: number;
  has_more: boolean;
}
export interface Board {
  project: Project;
  revision: number;
  columns: BoardColumn[];
}
