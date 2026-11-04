export interface Page<T> {
  items: T[];
  page: number;
  page_size: number;
  total: number;
}
export type TeamRole = "owner" | "admin" | "member";
export interface Team {
  id: string;
  name: string;
  description: string;
  owner_id: string;
  my_role: TeamRole;
  member_count: number;
  archived_at: string | null;
  created_at: string;
  updated_at: string;
}
export interface Member {
  user_id: string;
  nickname: string;
  email: string;
  role: TeamRole;
  status: "active" | "disabled";
  joined_at: string;
}
export interface Invite {
  id: string;
  team_id: string;
  expires_at: string;
  max_uses: number;
  used_count: number;
  revoked_at: string | null;
  created_by: string;
  created_at: string;
  status: "active" | "revoked" | "expired" | "exhausted";
}
export interface InviteCreated {
  invite: Invite;
  code: string;
}
export interface JoinedTeam {
  team: Team;
  already_joined: boolean;
}
export interface Activity {
  project_id: string | null;
  id: string;
  team_id: string;
  actor_id: string;
  actor_nickname: string;
  action: string;
  target_type: string;
  target_id: string;
  metadata: Record<string, unknown>;
  created_at: string;
}
