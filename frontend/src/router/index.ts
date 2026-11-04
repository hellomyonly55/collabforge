import { createRouter, createWebHistory } from "vue-router";
import { useAuthStore } from "../stores/auth";
import { safeRedirect } from "../utils/navigation";
export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/", redirect: "/teams" },
    {
      path: "/forbidden",
      component: () => import("../views/ForbiddenView.vue"),
    },
    {
      path: "/admin",
      component: () => import("../views/admin/AdminView.vue"),
      meta: { requiresAuth: true, requiresAdmin: true, section: "admin" },
    },
    {
      path: "/admin/users",
      component: () => import("../views/admin/AdminView.vue"),
      meta: { requiresAuth: true, requiresAdmin: true, section: "admin" },
    },
    {
      path: "/admin/teams",
      component: () => import("../views/admin/AdminView.vue"),
      meta: { requiresAuth: true, requiresAdmin: true, section: "admin" },
    },
    { path: "/health", component: () => import("../views/HealthView.vue") },
    {
      path: "/login",
      component: () => import("../views/auth/LoginView.vue"),
      meta: { guestOnly: true },
    },
    {
      path: "/register",
      component: () => import("../views/auth/RegisterView.vue"),
      meta: { guestOnly: true },
    },
    {
      path: "/connection-error",
      component: () => import("../views/ConnectionErrorView.vue"),
    },
    {
      path: "/teams",
      component: () => import("../views/teams/TeamWelcomeView.vue"),
      meta: { requiresAuth: true },
    },
    {
      path: "/teams/:teamId([1-9][0-9]{0,19})",
      component: () => import("../views/teams/TeamDetailView.vue"),
      meta: { requiresAuth: true },
    },
    {
      path: "/teams/:teamId([1-9][0-9]{0,19})/projects",
      component: () => import("../views/projects/ProjectListView.vue"),
      meta: { requiresAuth: true, section: "projects" },
    },
    {
      path: "/projects/:projectId([1-9][0-9]{0,19})",
      component: () => import("../views/projects/ProjectDetailView.vue"),
      meta: { requiresAuth: true, section: "projects" },
    },
    {
      path: "/projects/:projectId([1-9][0-9]{0,19})/board",
      component: () => import("../views/tasks/TaskBoardView.vue"),
      meta: { requiresAuth: true, section: "projects" },
    },
    {
      path: "/tasks/:taskId([1-9][0-9]{0,19})",
      component: () => import("../views/tasks/TaskDetailView.vue"),
      meta: { requiresAuth: true, section: "tasks" },
    },
    {
      path: "/teams/:teamId([1-9][0-9]{0,19})/activities",
      component: () => import("../views/activities/ActivityView.vue"),
      meta: { requiresAuth: true },
    },
    {
      path: "/projects/:projectId([1-9][0-9]{0,19})/activities",
      component: () => import("../views/activities/ActivityView.vue"),
      meta: { requiresAuth: true, section: "projects" },
    },
    {
      path: "/teams/:teamId([1-9][0-9]{0,19})/statistics",
      component: () => import("../views/statistics/StatisticsView.vue"),
      meta: { requiresAuth: true },
    },
    {
      path: "/projects/:projectId([1-9][0-9]{0,19})/statistics",
      component: () => import("../views/statistics/StatisticsView.vue"),
      meta: { requiresAuth: true, section: "projects" },
    },
    {
      path: "/my-tasks",
      component: () => import("../views/tasks/MyTasksView.vue"),
      meta: { requiresAuth: true, section: "tasks" },
    },
    {
      path: "/settings",
      component: () => import("../views/auth/SettingsView.vue"),
      meta: { requiresAuth: true },
    },
    { path: "/:pathMatch(.*)*", redirect: "/teams" },
  ],
});
router.beforeEach(async (to) => {
  const auth = useAuthStore();
  if (to.meta.requiresAuth || (to.meta.guestOnly && !auth.checked)) {
    try {
      await auth.refresh();
    } catch {
      return {
        path: "/connection-error",
        query: { redirect: safeRedirect(to.fullPath) },
      };
    }
  }
  if (to.meta.requiresAuth && !auth.user)
    return { path: "/login", query: { redirect: safeRedirect(to.fullPath) } };
  if (to.meta.requiresAdmin && auth.user?.platform_role !== "admin")
    return "/forbidden";
  if (to.meta.guestOnly && auth.user) return "/teams";
  return true;
});
