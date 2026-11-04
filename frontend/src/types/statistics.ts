export interface CountGroup {
  key: string;
  count: number;
}
export interface AssigneeStatistics {
  user_id: string | null;
  nickname: string;
  total: number;
  completed: number;
  overdue: number;
}
export interface Statistics {
  scope: "team" | "project";
  team_id: string;
  project_id: string | null;
  name: string;
  archived: boolean;
  as_of: string;
  today: string;
  timezone: string;
  total: number;
  completed: number;
  uncompleted: number;
  overdue: number;
  completion_rate: number;
  by_status: CountGroup[];
  by_priority: CountGroup[];
  by_assignee: AssigneeStatistics[];
}
