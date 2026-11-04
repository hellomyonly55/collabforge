import { request } from "./client";
import type { Envelope } from "../types/api";
import type { Page, Activity } from "../types/team";
import type { Project } from "../types/project";
export const projectAPI = {
  list: (teamID: string, page = 1, status = "all") =>
    request<Envelope<Page<Project>>>(
      `/teams/${teamID}/projects?page=${page}&page_size=20&status=${encodeURIComponent(status)}`,
    ),
  create: (teamID: string, name: string, description: string) =>
    request<Envelope<Project>>(`/teams/${teamID}/projects`, {
      method: "POST",
      body: { name, description },
    }),
  get: (id: string) => request<Envelope<Project>>(`/projects/${id}`),
  update: (id: string, name: string, description: string) =>
    request<Envelope<Project>>(`/projects/${id}`, {
      method: "PATCH",
      body: { name, description },
    }),
  archive: (id: string) =>
    request<void>(`/projects/${id}/archive`, { method: "POST" }),
  activities: (id: string, page = 1) =>
    request<Envelope<Page<Activity>>>(
      `/projects/${id}/activities?page=${page}&page_size=20`,
    ),
};
