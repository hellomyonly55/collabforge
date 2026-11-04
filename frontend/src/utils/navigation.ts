// Restrict post-login destinations to known internal routes.
export function safeRedirect(value: unknown): string {
  return typeof value === "string" &&
    ([
      "/teams",
      "/settings",
      "/my-tasks",
      "/admin",
      "/admin/users",
      "/admin/teams",
    ].includes(value) ||
      /^\/teams\/[1-9][0-9]{0,19}(?:\/(?:projects|activities|statistics))?$/.test(
        value,
      ) ||
      /^\/projects\/[1-9][0-9]{0,19}(?:\/(?:board|activities|statistics))?$/.test(
        value,
      ) ||
      /^\/tasks\/[1-9][0-9]{0,19}$/.test(value))
    ? value
    : "/teams";
}
