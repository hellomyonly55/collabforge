import { test, expect } from "@playwright/test";
import { randomUUID } from "node:crypto";

test("真实评论权限、并发冲突、分页及操作日志查询", async ({
  browser,
  context,
  page,
  baseURL,
}, testInfo) => {
  const author = await browser.newContext({ baseURL }),
    outside = await browser.newContext({ baseURL }),
    secondAuthor = await browser.newContext({ baseURL }),
    mobile = await browser.newContext({
      baseURL,
      viewport: { width: 390, height: 844 },
    });
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));
  try {
    const users = [];
    for (const [ctx, nickname] of [
      [context, "项目负责人"],
      [author, "协作成员"],
      [outside, "团队外用户"],
    ] as const) {
      const email = `comment-${randomUUID()}@example.test`,
        password = `TaskFlow-${randomUUID()}`;
      const register = await ctx.request.post("/api/v1/auth/register", {
        data: { email, password, nickname },
      });
      expect(register.status()).toBe(201);
      users.push((await register.json()).data);
      expect(
        (
          await ctx.request.post("/api/v1/auth/login", {
            data: { email, password },
          })
        ).status(),
      ).toBe(200);
    }
    const teamResponse = await context.request.post("/api/v1/teams", {
      data: { name: "TaskFlow 研发团队" },
    });
    expect(teamResponse.status()).toBe(201);
    const { data: team } = await teamResponse.json();
    const invite = await context.request.post(
      `/api/v1/teams/${team.id}/invites`,
      { data: {} },
    );
    expect(invite.status()).toBe(201);
    expect(
      (
        await author.request.post("/api/v1/teams/join", {
          data: { code: (await invite.json()).data.code },
        })
      ).status(),
    ).toBe(200);
    const projectResponse = await context.request.post(
      `/api/v1/teams/${team.id}/projects`,
      { data: { name: "评论与日志联调" } },
    );
    expect(projectResponse.status()).toBe(201);
    const { data: project } = await projectResponse.json();
    const taskResponse = await context.request.post(
      `/api/v1/projects/${project.id}/tasks`,
      {
        data: {
          title: "完善任务协作体验",
          description: "通过评论讨论需求，通过日志回顾变更。",
        },
      },
    );
    expect(taskResponse.status()).toBe(201);
    const { data: task } = await taskResponse.json();
    const authorPage = await author.newPage();
    await authorPage.goto(`/tasks/${task.id}`);
    const thread = authorPage.getByRole("region", {
      name: "任务评论",
      exact: true,
    });
    await expect(thread).toContainText("暂无评论");
    const xss =
      '需求已确认：保留讨论历史。<img src=x onerror="window.commentXSS=true">';
    await thread.getByLabel("发表评论", { exact: true }).fill(xss);
    await thread.getByRole("button", { name: "发表评论", exact: true }).click();
    await expect(thread.locator("[data-comment-id]")).toHaveCount(1);
    await expect(thread.locator(".comment-content")).toHaveText(xss);
    expect(
      await authorPage.evaluate(() => Reflect.get(window, "commentXSS")),
    ).toBeUndefined();
    await expect(thread.locator("img")).toHaveCount(0);
    await page.goto(`/projects/${project.id}/board`);
    await page.getByRole("button", { name: task.title, exact: true }).click();
    const drawer = page.getByRole("dialog");
    await expect(drawer).toHaveCSS("transform", "none");
    const ownerThread = drawer.getByRole("region", { name: "任务评论" });
    await expect(ownerThread.locator(".comment-content")).toHaveText(xss);
    await expect(
      ownerThread.getByRole("button", { name: "编辑评论", exact: true }),
    ).toHaveCount(0);
    await expect(
      ownerThread.getByRole("button", { name: "删除评论", exact: true }),
    ).toBeVisible();
    await thread.getByRole("button", { name: "编辑评论", exact: true }).click();
    await thread
      .getByLabel("编辑评论", { exact: true })
      .fill("接口与权限已核对，进入验收。");
    await thread.getByRole("button", { name: "保存评论", exact: true }).click();
    await expect(thread.locator(".comment-content")).toHaveText(
      "接口与权限已核对，进入验收。",
    );
    await secondAuthor.addCookies(await author.cookies());
    const secondPage = await secondAuthor.newPage();
    await secondPage.goto(`/tasks/${task.id}`);
    const secondThread = secondPage.getByRole("region", { name: "任务评论" });
    await secondThread
      .getByRole("button", { name: "编辑评论", exact: true })
      .click();
    await secondThread
      .getByLabel("编辑评论", { exact: true })
      .fill("旧窗口修改");
    await thread.getByRole("button", { name: "编辑评论", exact: true }).click();
    await thread
      .getByLabel("编辑评论", { exact: true })
      .fill("最新验收结论：功能通过。");
    await thread.getByRole("button", { name: "保存评论", exact: true }).click();
    await expect(thread.locator(".comment-content")).toHaveText(
      "最新验收结论：功能通过。",
    );
    await secondThread
      .getByRole("button", { name: "保存评论", exact: true })
      .click();
    await expect(secondThread.getByRole("alert")).toContainText(
      "评论已被其他操作修改",
    );
    await expect(secondThread.locator(".comment-content")).toHaveText(
      "最新验收结论：功能通过。",
    );
    await expect(
      secondThread.getByLabel("编辑评论", { exact: true }),
    ).toHaveCount(0);
    await ownerThread
      .getByRole("button", { name: "刷新评论", exact: true })
      .click();
    await expect(ownerThread.locator(".comment-content")).toHaveText(
      "最新验收结论：功能通过。",
    );
    await ownerThread.locator(".comment-content").scrollIntoViewIfNeeded();
    await page.screenshot({
      path: testInfo.outputPath("comments-drawer.png"),
      animations: "disabled",
    });
    page.once("dialog", (d) => d.accept());
    await ownerThread
      .getByRole("button", { name: "删除评论", exact: true })
      .click();
    await expect(ownerThread).toContainText("暂无评论");
    await authorPage.reload();
    await expect(
      authorPage.getByRole("region", { name: "任务评论" }),
    ).toContainText("暂无评论");
    const stored = await context.request.get(
      `/api/v1/tasks/${task.id}/comments`,
    );
    expect(stored.status()).toBe(200);
    expect((await stored.json()).data.total).toBe(0);
    for (let i = 0; i < 23; i++) {
      const response = await author.request.post(
        `/api/v1/tasks/${task.id}/comments`,
        { data: { content: `验收记录 ${i + 1}：真实接口与数据库已联通。` } },
      );
      expect(response.status()).toBe(201);
    }
    await authorPage.reload();
    const paged = authorPage.getByRole("region", { name: "任务评论" });
    await expect(paged.locator("[data-comment-id]")).toHaveCount(20);
    await paged.getByRole("button", { name: "下一页", exact: true }).click();
    await expect(paged.locator("[data-comment-id]")).toHaveCount(3);
    await page.goto(`/projects/${project.id}/activities`);
    await expect(
      page.getByRole("heading", { name: "操作日志", exact: true }),
    ).toBeVisible();
    await expect(page.locator("[data-activity-id]")).toHaveCount(20);
    await page
      .getByLabel("操作类型", { exact: true })
      .selectOption("comment.updated");
    await page.getByLabel("目标类型", { exact: true }).selectOption("comment");
    await page.getByLabel("操作者 ID", { exact: true }).fill(users[1].id);
    const today = new Intl.DateTimeFormat("en-CA", {
      timeZone: "Asia/Shanghai",
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
    }).format(new Date());
    await page.getByLabel("开始日期", { exact: true }).fill(today);
    await page.getByLabel("结束日期", { exact: true }).fill(today);
    await page.getByRole("button", { name: "查询日志", exact: true }).click();
    await expect(page.locator("[data-activity-id]")).toHaveCount(2);
    await expect(page.locator(".activity-list")).toContainText("编辑了评论");
    await expect(page.locator(".activity-list")).not.toContainText(
      "最新验收结论",
    );
    await expect(page.locator(".activity-list")).toContainText(task.title);
    await page.screenshot({
      path: testInfo.outputPath("activity-project.png"),
      fullPage: true,
      animations: "disabled",
    });
    await page.getByRole("button", { name: "重置筛选", exact: true }).click();
    await expect(page.locator("[data-activity-id]")).toHaveCount(20);
    await page.getByRole("button", { name: "下一页", exact: true }).click();
    await expect(page.locator("[data-activity-id]")).toHaveCount(9);
    await page.goto(`/teams/${team.id}/activities`);
    await page.getByLabel("项目 ID", { exact: true }).fill(project.id);
    await page
      .getByLabel("操作类型", { exact: true })
      .selectOption("task.created");
    await page.getByRole("button", { name: "查询日志", exact: true }).click();
    await expect(page.locator("[data-activity-id]")).toHaveCount(1);
    await page.getByRole("link", { name: "查看任务 ↗", exact: true }).click();
    await expect(page.getByLabel("任务标题", { exact: true })).toHaveValue(
      task.title,
    );
    const outsidePage = await outside.newPage();
    await outsidePage.goto(`/projects/${project.id}/activities`);
    await expect(outsidePage.getByRole("alert")).toContainText("没有访问权限");
    await outsidePage.goto(`/tasks/${task.id}`);
    await expect(outsidePage.getByRole("alert")).toContainText("没有访问权限");
    await expect(
      outsidePage.getByRole("region", { name: "任务评论" }),
    ).toHaveCount(0);
    expect(
      (
        await context.request.post(`/api/v1/projects/${project.id}/archive`)
      ).status(),
    ).toBe(204);
    await authorPage.reload();
    await expect(
      authorPage.getByRole("region", { name: "任务评论" }),
    ).toContainText("评论仅可查看");
    await expect(
      authorPage.getByRole("button", { name: "发表评论", exact: true }),
    ).toHaveCount(0);
    await expect(authorPage.locator(".comment-actions")).toHaveCount(0);
    await mobile.addCookies(await author.cookies());
    const mobilePage = await mobile.newPage();
    await mobilePage.goto(`/tasks/${task.id}`);
    await expect(mobilePage.locator("[data-comment-id]")).toHaveCount(20);
    expect(
      await mobilePage.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    await mobilePage
      .locator(".comment-content")
      .first()
      .scrollIntoViewIfNeeded();
    await mobilePage.screenshot({
      path: testInfo.outputPath("comments-mobile.png"),
      animations: "disabled",
    });
    await mobilePage.goto(`/projects/${project.id}/activities`);
    await expect(mobilePage.locator("[data-activity-id]")).toHaveCount(20);
    expect(
      await mobilePage.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    await mobilePage
      .locator("[data-activity-id]")
      .first()
      .scrollIntoViewIfNeeded();
    await mobilePage.screenshot({
      path: testInfo.outputPath("activity-mobile.png"),
      animations: "disabled",
    });
    expect(errors).toEqual([]);
  } finally {
    await author.close();
    await outside.close();
    await secondAuthor.close();
    await mobile.close();
  }
});
