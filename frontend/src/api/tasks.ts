import { request } from "./client";
import type { Envelope } from "../types/api";
import type {
  Task,
  TaskFields,
  TaskFilters,
  TaskStatus,
  TaskPage,
  Board,
} from "../types/task";
export function taskQuery(
  filters: Partial<TaskFilters> = {},
  page?: number,
  pageSize?: number,
  revision?: number,
) {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(filters)) {
    if (value) query.set(key, value);
  }
  if (page !== undefined) query.set("page", String(page));
  if (pageSize !== undefined) query.set("page_size", String(pageSize));
  if (revision !== undefined) query.set("revision", String(revision));
  return query.toString();
}
export const taskAPI = {
  board: (projectID: string, filters: Partial<TaskFilters>) =>
    request<Envelope<Board>>(
      `/projects/${projectID}/board?${taskQuery(filters)}`,
    ),
  list: (
    projectID: string,
    filters: Partial<TaskFilters> = {},
    page = 1,
    pageSize = 20,
    revision?: number,
  ) =>
    request<Envelope<TaskPage>>(
      `/projects/${projectID}/tasks?${taskQuery(filters, page, pageSize, revision)}`,
    ),
  mine: (filters: Partial<TaskFilters>, page = 1) =>
    request<Envelope<TaskPage>>(`/me/tasks?${taskQuery(filters, page, 20)}`),
  get: (id: string) => request<Envelope<Task>>(`/tasks/${id}`),
  create: (projectID: string, fields: TaskFields) =>
    request<Envelope<Task>>(`/projects/${projectID}/tasks`, {
      method: "POST",
      body: fields,
    }),
  update: (id: string, version: number, fields: TaskFields) =>
    request<Envelope<Task>>(`/tasks/${id}`, {
      method: "PATCH",
      body: { ...fields, version },
    }),
  move: (
    id: string,
    version: number,
    status: TaskStatus,
    before_task_id: string | null,
  ) =>
    request<Envelope<Task>>(`/tasks/${id}/move`, {
      method: "POST",
      body: { version, status, before_task_id },
    }),
  delete: (id: string, version: number) =>
    request<void>(`/tasks/${id}?version=${version}`, { method: "DELETE" }),
};
