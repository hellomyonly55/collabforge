import type { TeamRole } from "./team";
export interface Project {
  id: string;
  team_id: string;
  team_name: string;
  my_role: TeamRole;
  team_archived_at: string | null;
  name: string;
  description: string;
  created_by: string;
  creator_nickname: string;
  archived_at: string | null;
  created_at: string;
  updated_at: string;
}
