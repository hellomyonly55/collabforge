import type { TaskFilters, TaskStatus, TaskPriority } from "../types/task";
export const statuses: TaskStatus[] = [
  "todo",
  "in_progress",
  "in_review",
  "done",
];
export const statusLabels: Record<TaskStatus, string> = {
  todo: "待办",
  in_progress: "进行中",
  in_review: "待验收",
  done: "已完成",
};
export const priorityLabels: Record<TaskPriority, string> = {
  low: "低",
  medium: "中",
  high: "高",
  urgent: "紧急",
};
export const emptyTaskFilters = (): TaskFilters => ({
  keyword: "",
  status: "",
  priority: "",
  assignee_id: "",
  overdue: "",
});
