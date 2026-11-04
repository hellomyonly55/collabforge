import { request } from "./client";
import type { Envelope } from "../types/api";
import type { Activity, Page } from "../types/team";
export interface ActivityFilters {
  action: string;
  target_type: string;
  actor_id: string;
  project_id: string;
  from: string;
  to: string;
}
export const emptyActivityFilters = (): ActivityFilters => ({
  action: "",
  target_type: "",
  actor_id: "",
  project_id: "",
  from: "",
  to: "",
});
export function listActivities(
  scope: "teams" | "projects",
  id: string,
  filters: ActivityFilters,
  page: number,
) {
  const q = new URLSearchParams({ page: String(page), page_size: "20" });
  for (const [key, value] of Object.entries(filters)) {
    if (!value) continue;
    const date =
      key === "from"
        ? `${value}T00:00:00+08:00`
        : key === "to"
          ? `${value}T23:59:59.999999+08:00`
          : value;
    q.set(key, date);
  }
  return request<Envelope<Page<Activity>>>(`/${scope}/${id}/activities?${q}`);
}
