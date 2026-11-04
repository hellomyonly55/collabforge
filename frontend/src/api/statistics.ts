import { request } from "./client";
import type { Envelope } from "../types/api";
import type { Statistics } from "../types/statistics";
export const statisticsAPI = {
  get: (scope: "teams" | "projects", id: string) =>
    request<Envelope<Statistics>>(`/${scope}/${id}/statistics`),
};
