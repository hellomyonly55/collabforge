import type { TeamRole } from "../types/team";
export const roleLabels: Record<TeamRole, string> = {
  owner: "所有者",
  admin: "管理员",
  member: "成员",
};
export const inviteLabels = {
  active: "有效",
  revoked: "已撤销",
  expired: "已过期",
  exhausted: "已用尽",
};
export const activityLabels: Record<string, string> = {
  "comment.created": "发表了评论",
  "comment.updated": "编辑了评论",
  "comment.deleted": "删除了评论",
  "task.created": "创建了任务",
  "task.updated": "更新了任务资料",
  "task.moved": "移动了任务",
  "task.deleted": "删除了任务",
  "task.unassigned": "清空了离队成员的任务分配",
  "project.created": "创建了项目",
  "project.updated": "更新了项目资料",
  "project.archived": "归档了项目",
  "team.created": "创建了团队",
  "team.updated": "更新了团队资料",
  "team.archived": "归档了团队",
  "team.ownership_transferred": "转让了团队所有权",
  "member.joined": "加入了团队",
  "member.left": "退出了团队",
  "member.removed": "移除了成员",
  "member.role_changed": "更改了成员角色",
  "invite.created": "创建了邀请码",
  "invite.revoked": "撤销了邀请码",
};
export const dateTime = (value: string) =>
  new Date(value).toLocaleString("zh-CN", { hour12: false });
