export interface AdminUser {
  id: string;
  email: string;
  nickname: string;
  platform_role: "admin" | "user";
  status: "active" | "disabled";
  owned_team_count: number;
  created_at: string;
  updated_at: string;
}
export interface AdminTeam {
  id: string;
  name: string;
  description: string;
  owner_id: string;
  owner_nickname: string;
  owner_email: string;
  member_count: number;
  project_count: number;
  task_count: number;
  archived_at: string | null;
  created_at: string;
}
export interface AdminStatistics {
  users: number;
  active_users: number;
  disabled_users: number;
  admins: number;
  teams: number;
  archived_teams: number;
  projects: number;
  archived_projects: number;
  tasks: number;
  completed_tasks: number;
  overdue_tasks: number;
  comments: number;
  as_of: string;
  today: string;
  timezone: string;
}
