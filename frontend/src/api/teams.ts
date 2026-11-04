import { request } from "./client";
import type { Envelope } from "../types/api";
import type {
  Page,
  Team,
  Member,
  Invite,
  InviteCreated,
  JoinedTeam,
  Activity,
} from "../types/team";
const query = (page: number) => `page=${page}&page_size=20`;
export const teamAPI = {
  list: (page = 1, status = "all") =>
    request<Envelope<Page<Team>>>(
      `/teams?${query(page)}&status=${encodeURIComponent(status)}`,
    ),
  create: (name: string, description: string) =>
    request<Envelope<Team>>("/teams", {
      method: "POST",
      body: { name, description },
    }),
  get: (id: string) => request<Envelope<Team>>(`/teams/${id}`),
  update: (id: string, name: string, description: string) =>
    request<Envelope<Team>>(`/teams/${id}`, {
      method: "PATCH",
      body: { name, description },
    }),
  join: (code: string) =>
    request<Envelope<JoinedTeam>>("/teams/join", {
      method: "POST",
      body: { code },
    }),
  members: (id: string, page = 1, pageSize = 20) =>
    request<Envelope<Page<Member>>>(`/teams/${id}/members?page=${page}&page_size=${pageSize}`),
  role: (id: string, userID: string, role: "admin" | "member") =>
    request<void>(`/teams/${id}/members/${userID}`, {
      method: "PATCH",
      body: { role },
    }),
  remove: (id: string, userID: string) =>
    request<void>(`/teams/${id}/members/${userID}`, { method: "DELETE" }),
  leave: (id: string) =>
    request<void>(`/teams/${id}/leave`, { method: "POST" }),
  transfer: (id: string, userID: string) =>
    request<Envelope<Team>>(`/teams/${id}/transfer-ownership`, {
      method: "POST",
      body: { user_id: userID },
    }),
  archive: (id: string) =>
    request<void>(`/teams/${id}/archive`, { method: "POST" }),
  invites: (id: string, page = 1) =>
    request<Envelope<Page<Invite>>>(`/teams/${id}/invites?${query(page)}`),
  createInvite: (id: string, expires_in_days: number, max_uses: number) =>
    request<Envelope<InviteCreated>>(`/teams/${id}/invites`, {
      method: "POST",
      body: { expires_in_days, max_uses },
    }),
  revoke: (id: string, inviteID: string) =>
    request<void>(`/teams/${id}/invites/${inviteID}`, { method: "DELETE" }),
  activities: (id: string, page = 1) =>
    request<Envelope<Page<Activity>>>(`/teams/${id}/activities?${query(page)}`),
};
