import {
  test,
  expect,
  type Browser,
  type BrowserContext,
} from "@playwright/test";
import { randomUUID } from "node:crypto";
async function taskFixture(
  browser: Browser,
  owner: BrowserContext,
  baseURL: string | undefined,
) {
  const member = await browser.newContext({ baseURL }),
    outside = await browser.newContext({ baseURL }),
    password = `Task-${randomUUID()}`;
  const users = [];
  for (const [ctx, nickname] of [
    [owner, "任务负责人"],
    [member, "协作伙伴"],
    [outside, "团队外账号"],
  ] as const) {
    const email = `task-${randomUUID()}@example.test`;
    const response = await ctx.request.post("/api/v1/auth/register", {
      data: { email, password, nickname },
    });
    expect(response.status()).toBe(201);
    const { data: user } = await response.json();
    expect(
      (
        await ctx.request.post("/api/v1/auth/login", {
          data: { email, password },
        })
      ).status(),
    ).toBe(200);
    users.push({ ...user, email });
  }
  const response = await owner.request.post("/api/v1/teams", {
    data: { name: "TaskFlow 协作小组" },
  });
  expect(response.status()).toBe(201);
  const { data: team } = await response.json();
  const invited = await owner.request.post(`/api/v1/teams/${team.id}/invites`, {
    data: {},
  });
  expect(invited.status()).toBe(201);
  const { data: invite } = await invited.json();
  expect(
    (
      await member.request.post("/api/v1/teams/join", {
        data: { code: invite.code },
      })
    ).status(),
  ).toBe(200);
  const created = await owner.request.post(
    `/api/v1/teams/${team.id}/projects`,
    {
      data: {
        name: "TaskFlow 开发迭代",
        description: "用真实任务推进协作平台开发。",
      },
    },
  );
  expect(created.status()).toBe(201);
  const { data: project } = await created.json();
  return {
    member,
    outside,
    users,
    team,
    project,
    boardPath: `/projects/${project.id}/board`,
  };
}

test("真实任务创建、拖拽、编辑、筛选、我的任务和软删除", async ({
  browser,
  context,
  page,
  baseURL,
}, testInfo) => {
  const f = await taskFixture(browser, context, baseURL),
    title = "完成真实任务看板",
    past = new Date(Date.now() - 3 * 86400000).toISOString().slice(0, 10);
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));
  try {
    await page.goto(`/projects/${f.project.id}`);
    await page.getByRole("link", { name: "任务看板", exact: true }).click();
    await expect(
      page.getByRole("heading", { name: "任务看板", exact: true }),
    ).toBeVisible();
    await page.getByRole("button", { name: "创建任务", exact: true }).click();
    const drawer = page.getByRole("dialog");
    await drawer.getByLabel("任务标题", { exact: true }).fill(title);
    await drawer
      .getByLabel("任务描述", { exact: true })
      .fill("实现真实后端、错误处理和团队权限校验。");
    await drawer.getByLabel("优先级", { exact: true }).selectOption("high");
    await drawer.getByLabel("截止日期", { exact: true }).fill(past);
    await drawer
      .getByLabel("负责人", { exact: true })
      .selectOption(f.users[1]!.id);
    const createdResponse = page.waitForResponse(
      (r) =>
        r.url().endsWith(`/api/v1/projects/${f.project.id}/tasks`) &&
        r.request().method() === "POST",
    );
    await drawer.getByRole("button", { name: "确认创建任务" }).click();
    const { data: task } = await (await createdResponse).json();
    await expect(drawer).toHaveCount(0);
    const card = () => page.locator(`[data-task-id="${task.id}"]`);
    await expect(card()).toContainText(title);
    await expect(card()).toContainText("协作伙伴");
    await expect(card()).toContainText("逾期");
    const moved = page.waitForResponse(
      (r) =>
        r.url().endsWith(`/tasks/${task.id}/move`) &&
        r.request().method() === "POST",
    );
    await card().dragTo(
      page.locator('[data-status="in_progress"] .column-body'),
    );
    expect((await moved).status()).toBe(200);
    await expect(
      page.locator(`[data-status="in_progress"] [data-task-id="${task.id}"]`),
    ).toBeVisible();
    await page.reload();
    await expect(
      page.locator(`[data-status="in_progress"] [data-task-id="${task.id}"]`),
    ).toBeVisible();
    await page.screenshot({
      path: testInfo.outputPath("task-board.png"),
      fullPage: true,
      animations: "disabled",
    });
    const memberPage = await f.member.newPage();
    memberPage.on("pageerror", (e) => errors.push(e.message));
    await memberPage.goto(f.boardPath);
    await memberPage.getByRole("button", { name: title, exact: true }).click();
    const memberDrawer = memberPage.getByRole("dialog");
    await expect(
      memberDrawer.getByRole("button", { name: "删除任务", exact: true }),
    ).toHaveCount(0);
    await memberDrawer
      .getByLabel("任务描述", { exact: true })
      .fill("协作伙伴补充了验收要求。");
    await memberDrawer.getByRole("button", { name: "保存任务资料" }).click();
    await expect(memberDrawer.getByRole("alert")).toContainText(
      "任务资料已保存",
    );
    await memberPage.keyboard.press("Escape");
    await expect(memberDrawer).toHaveCount(0);
    await memberPage
      .getByRole("link", { name: "我的任务", exact: true })
      .click();
    await expect(
      memberPage.getByRole("link", { name: title, exact: true }),
    ).toBeVisible();
    await memberPage
      .getByLabel("截止情况", { exact: true })
      .selectOption("true");
    await memberPage.getByRole("button", { name: "筛选", exact: true }).click();
    await expect(
      memberPage.getByRole("link", { name: title, exact: true }),
    ).toBeVisible();
    await memberPage.getByRole("link", { name: title, exact: true }).click();
    await memberPage
      .getByLabel("移动到", { exact: true })
      .selectOption("in_review");
    await memberPage.getByRole("button", { name: "移到该列末尾" }).click();
    await expect(memberPage.getByRole("alert")).toContainText("任务状态已更新");
    await page.reload();
    await expect(
      page.locator(`[data-status="in_review"] [data-task-id="${task.id}"]`),
    ).toBeVisible();
    await page.getByLabel("关键词", { exact: true }).fill("不存在的任务");
    await page.getByRole("button", { name: "筛选", exact: true }).click();
    await expect(card()).toHaveCount(0);
    await page.getByRole("button", { name: "重置", exact: true }).click();
    await expect(card()).toBeVisible();
    await page.getByRole("button", { name: "列表", exact: true }).click();
    await expect(page.locator("tbody")).toContainText(title);
    await page.getByRole("button", { name: title, exact: true }).click();
    await expect(drawer.getByLabel("任务标题", { exact: true })).toHaveValue(title);
    await expect(drawer).toHaveCSS("transform", "none");
    await page.screenshot({
      path: testInfo.outputPath("task-drawer.png"),
      fullPage: true,
      animations: "disabled",
    });
    await page.keyboard.press("Escape");
    const outsidePage = await f.outside.newPage();
    await outsidePage.goto(`/tasks/${task.id}`);
    await expect(outsidePage.getByRole("alert")).toContainText("没有访问权限");
    await memberPage.goto(f.boardPath);
    await memberPage
      .getByRole("button", { name: "创建任务", exact: true })
      .click();
    await memberDrawer
      .getByLabel("任务标题", { exact: true })
      .fill("可删除的验收任务");
    await memberDrawer.getByRole("button", { name: "确认创建任务" }).click();
    await expect(memberDrawer).toHaveCount(0);
    await memberPage
      .getByRole("button", { name: "可删除的验收任务", exact: true })
      .click();
    memberPage.once("dialog", (dialog) => dialog.accept());
    await memberDrawer
      .getByRole("button", { name: "删除任务", exact: true })
      .click();
    await expect(memberDrawer).toHaveCount(0);
    await expect(
      memberPage.getByRole("button", { name: "可删除的验收任务", exact: true }),
    ).toHaveCount(0);
    expect(
      (
        await context.request.post(`/api/v1/projects/${f.project.id}/archive`)
      ).status(),
    ).toBe(204);
    await page.goto(f.boardPath);
    await expect(
      page.getByText("团队或项目已归档，任务仅可查看"),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "创建任务", exact: true }),
    ).toHaveCount(0);
    await expect(card()).toHaveAttribute("draggable", "false");
    expect(errors).toEqual([]);
  } finally {
    await f.member.close();
    await f.outside.close();
  }
});

test("双会话版本冲突提示与移除成员后的任务清理", async ({
  browser,
  context,
  page,
  baseURL,
}) => {
  const f = await taskFixture(browser, context, baseURL);
  try {
    const response = await context.request.post(
      `/api/v1/projects/${f.project.id}/tasks`,
      { data: { title: "并发协作任务", assignee_id: f.users[1]!.id } },
    );
    expect(response.status()).toBe(201);
    const { data: task } = await response.json();
    const other = await f.member.newPage();
    await page.goto(f.boardPath);
    await other.goto(f.boardPath);
    await page
      .getByRole("button", { name: "并发协作任务", exact: true })
      .click();
    await other
      .getByRole("button", { name: "并发协作任务", exact: true })
      .click();
    const drawer = page.getByRole("dialog"),
      otherDrawer = other.getByRole("dialog");
    await otherDrawer
      .getByLabel("任务描述", { exact: true })
      .fill("另一个会话先保存的内容");
    await otherDrawer.getByRole("button", { name: "保存任务资料" }).click();
    await expect(otherDrawer.getByRole("alert")).toContainText(
      "任务资料已保存",
    );
    await drawer.getByLabel("任务标题", { exact: true }).fill("旧版本尝试覆盖");
    await drawer.getByRole("button", { name: "保存任务资料" }).click();
    await expect(drawer.getByRole("alert")).toContainText(
      "任务已被其他操作修改",
    );
    await expect(drawer.getByLabel("任务标题", { exact: true })).toHaveValue(
      "并发协作任务",
    );
    await expect(drawer.getByLabel("任务描述", { exact: true })).toHaveValue(
      "另一个会话先保存的内容",
    );
    await drawer
      .getByLabel("任务标题", { exact: true })
      .fill("检查冲突后重新保存");
    await drawer.getByRole("button", { name: "保存任务资料" }).click();
    await expect(drawer.getByRole("alert")).toContainText("任务资料已保存");
    await page.keyboard.press("Escape");
    await page.goto(`/teams/${f.team.id}`);
    await page.getByRole("button", { name: "成员", exact: true }).click();
    const row = page.locator("tbody tr").filter({ hasText: f.users[1]!.email });
    page.once("dialog", (dialog) => dialog.accept());
    await row.getByRole("button", { name: "移除", exact: true }).click();
    await expect(row).toHaveCount(0);
    await other.goto("/my-tasks");
    await expect(other.locator("tbody")).toContainText(
      "暂无分配给你的符合条件的任务",
    );
    const current = await context.request.get(`/api/v1/tasks/${task.id}`);
    expect(current.status()).toBe(200);
    expect((await current.json()).data.assignee_id).toBeNull();
    await page.goto(f.boardPath);
    await expect(page.locator(`[data-task-id="${task.id}"]`)).toContainText(
      "未分配",
    );
  } finally {
    await f.member.close();
    await f.outside.close();
  }
});

test("每列分页遇到并发变更自动刷新，移动端可使用列表与任务详情", async ({
  browser,
  context,
  page,
  baseURL,
}, testInfo) => {
  const f = await taskFixture(browser, context, baseURL);
  try {
    const tasks = [];
    for (let i = 0; i < 55; i++) {
      const response = await context.request.post(
        `/api/v1/projects/${f.project.id}/tasks`,
        { data: { title: `分页任务 ${String(i).padStart(2, "0")}` } },
      );
      expect(response.status()).toBe(201);
      tasks.push((await response.json()).data);
    }
    await page.goto(f.boardPath);
    await expect(page.locator("[data-task-id]")).toHaveCount(50);
    expect(
      (
        await context.request.patch(`/api/v1/tasks/${tasks[0].id}`, {
          data: { version: tasks[0].version, title: "分页期间任务已更新" },
        })
      ).status(),
    ).toBe(200);
    await page
      .getByRole("button", { name: "加载更多（50 / 55）", exact: true })
      .click();
    await expect(page.getByRole("alert")).toContainText("看板已变化");
    await expect(page.locator("[data-task-id]")).toHaveCount(50);
    await page
      .getByRole("button", { name: "加载更多（50 / 55）", exact: true })
      .click();
    await expect(page.locator("[data-task-id]")).toHaveCount(55);
    const ids = await page
      .locator("[data-task-id]")
      .evaluateAll((elements) =>
        elements.map((e) => e.getAttribute("data-task-id")),
      );
    expect(new Set(ids).size).toBe(55);
    const mobile = await browser.newContext({
      baseURL,
      viewport: { width: 390, height: 844 },
    });
    try {
      await mobile.addCookies(await context.cookies());
      const mobilePage = await mobile.newPage();
      await mobilePage.goto(f.boardPath);
      await expect(mobilePage.locator("[data-task-id]")).toHaveCount(50);
      expect(
        await mobilePage.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
      ).toBe(true);
      await mobilePage
        .getByRole("button", { name: "列表", exact: true })
        .click();
      await expect(mobilePage.locator("tbody tr")).toHaveCount(20);
      await mobilePage.screenshot({
        path: testInfo.outputPath("task-list-mobile.png"),
        fullPage: true,
        animations: "disabled",
      });
      await mobilePage.goto(`/tasks/${tasks[0].id}`);
      await expect(
        mobilePage.getByLabel("任务标题", { exact: true }),
      ).toHaveValue("分页期间任务已更新");
      expect(
        await mobilePage.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
      ).toBe(true);
      await mobilePage.screenshot({
        path: testInfo.outputPath("task-detail-mobile.png"),
        fullPage: true,
        animations: "disabled",
      });
    } finally {
      await mobile.close();
    }
  } finally {
    await f.member.close();
    await f.outside.close();
  }
});
