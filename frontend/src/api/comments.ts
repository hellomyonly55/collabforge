import { request } from "./client";
import type { Envelope } from "../types/api";
import type { Page } from "../types/team";
import type { Comment } from "../types/comment";
export const commentAPI = {
  list: (taskID: string, page = 1) =>
    request<Envelope<Page<Comment>>>(
      `/tasks/${taskID}/comments?page=${page}&page_size=20`,
    ),
  create: (taskID: string, content: string) =>
    request<Envelope<Comment>>(`/tasks/${taskID}/comments`, {
      method: "POST",
      body: { content },
    }),
  update: (id: string, content: string, version: number) =>
    request<Envelope<Comment>>(`/comments/${id}`, {
      method: "PATCH",
      body: { content, version },
    }),
  remove: (id: string, version: number) =>
    request<void>(`/comments/${id}?version=${version}`, { method: "DELETE" }),
};
