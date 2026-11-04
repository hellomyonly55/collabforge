import { request } from "./client";
import type { Envelope } from "../types/api";
import type { Page } from "../types/team";
import type { AdminUser, AdminTeam, AdminStatistics } from "../types/admin";
export const adminAPI = {
  statistics: () => request<Envelope<AdminStatistics>>("/admin/statistics"),
  users: (page: number, keyword: string, status: string, role: string) =>
    request<Envelope<Page<AdminUser>>>(
      `/admin/users?${new URLSearchParams({ page: String(page), page_size: "20", keyword, status, role })}`,
    ),
  teams: (page: number, keyword: string, status: string) =>
    request<Envelope<Page<AdminTeam>>>(
      `/admin/teams?${new URLSearchParams({ page: String(page), page_size: "20", keyword, status })}`,
    ),
  status: (id: string, status: string, reason: string) =>
    request<Envelope<AdminUser>>(`/admin/users/${id}/status`, {
      method: "PATCH",
      body: { status, reason },
    }),
  archive: (id: string, reason: string) =>
    request<void>(`/admin/teams/${id}/archive`, {
      method: "POST",
      body: { reason },
    }),
};
