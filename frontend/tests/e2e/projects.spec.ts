import { test, expect } from "@playwright/test";
import { randomUUID } from "node:crypto";

test("真实项目创建、编辑、共享访问、归档与团队隔离", async ({
  page,
  context,
  browser,
  baseURL,
}, testInfo) => {
  const password = `Project-${randomUUID()}`,
    teamName = `项目验收团队-${randomUUID().slice(0, 8)}`,
    projectName = `TaskFlow 研发计划-${randomUUID().slice(0, 8)}`;
  const member = await browser.newContext({ baseURL }),
    outside = await browser.newContext({ baseURL });
  const emails = [
    `pm-${randomUUID()}@example.test`,
    `dev-${randomUUID()}@example.test`,
    `outside-${randomUUID()}@example.test`,
  ];
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));
  try {
    for (const [ctx, email, nickname] of [
      [context, emails[0], "项目负责人"],
      [member, emails[1], "协作开发者"],
      [outside, emails[2], "其他团队成员"],
    ] as const) {
      expect(
        (
          await ctx.request.post("/api/v1/auth/register", {
            data: { email, password, nickname },
          })
        ).status(),
      ).toBe(201);
      expect(
        (
          await ctx.request.post("/api/v1/auth/login", {
            data: { email, password },
          })
        ).status(),
      ).toBe(200);
    }
    const teamResponse = await context.request.post("/api/v1/teams", {
      data: { name: teamName },
    });
    expect(teamResponse.status()).toBe(201);
    const { data: team } = await teamResponse.json();
    const inviteResponse = await context.request.post(
      `/api/v1/teams/${team.id}/invites`,
      { data: {} },
    );
    expect(inviteResponse.status()).toBe(201);
    const { data: invite } = await inviteResponse.json();
    expect(
      (
        await member.request.post("/api/v1/teams/join", {
          data: { code: invite.code },
        })
      ).status(),
    ).toBe(200);
    await page.goto(`/teams/${team.id}`);
    await page.getByRole("link", { name: "项目", exact: true }).click();
    await expect(
      page.getByRole("heading", { name: "团队项目", exact: true }),
    ).toBeVisible();
    await page.getByRole("button", { name: "创建项目", exact: true }).click();
    await page.getByLabel("项目名称", { exact: true }).fill(projectName);
    await page
      .getByLabel("项目简介", { exact: true })
      .fill("通过真实 API 开发团队协作平台，完成前后端联调与验收。");
    await page.getByRole("button", { name: "确认创建项目" }).click();
    await expect(page).toHaveURL(new RegExp("/projects/[0-9]+$"));
    const projectPath = new URL(page.url()).pathname,
      projectID = projectPath.split("/").pop()!;
    await expect(
      page.getByRole("heading", { name: projectName, exact: true }),
    ).toBeVisible();
    await page
      .getByLabel("项目名称", { exact: true })
      .fill(projectName + " · 第一迭代");
    await page
      .getByLabel("项目简介", { exact: true })
      .fill("第一迭代：账号、团队与项目管理。");
    await page.getByRole("button", { name: "保存项目资料" }).click();
    await expect(page.getByRole("alert")).toContainText("项目资料已保存");
    await page.reload();
    await expect(page.getByLabel("项目名称", { exact: true })).toHaveValue(
      projectName + " · 第一迭代",
    );
    await page.screenshot({
      path: testInfo.outputPath("project-detail.png"),
      fullPage: true,
      animations: "disabled",
    });
    const memberPage = await member.newPage();
    memberPage.on("pageerror", (e) => errors.push(e.message));
    await memberPage.goto(projectPath);
    await expect(
      memberPage.getByRole("heading", {
        name: projectName + " · 第一迭代",
        exact: true,
      }),
    ).toBeVisible();
    await expect(
      memberPage.getByRole("button", { name: "保存项目资料" }),
    ).toHaveCount(0);
    await expect(
      memberPage.getByRole("button", { name: "归档项目", exact: true }),
    ).toHaveCount(0);
    await memberPage.goto(`/teams/${team.id}/projects`);
    await expect(
      memberPage.getByRole("button", { name: "创建项目", exact: true }),
    ).toHaveCount(0);
    await expect(
      memberPage.getByRole("link", {
        name: `进入项目 ${projectName} · 第一迭代`,
      }),
    ).toBeVisible();
    const outsidePage = await outside.newPage();
    await outsidePage.goto(projectPath);
    await expect(outsidePage.getByRole("alert")).toContainText("没有访问权限");
    expect(
      (await outside.request.get(`/api/v1/teams/${team.id}/projects`)).status(),
    ).toBe(404);
    await page.getByRole("button", { name: "项目动态", exact: true }).click();
    await expect(page.locator(".activity-list")).toContainText("创建了项目");
    await expect(page.locator(".activity-list")).toContainText(
      "更新了项目资料",
    );
    await page.goto(`/teams/${team.id}`);
    await page.getByRole("button", { name: "团队动态", exact: true }).click();
    await expect(page.locator(".activity-list")).toContainText("创建了项目");
    await page.getByRole("link", { name: "查看项目 ↗" }).first().click();
    await expect(page).toHaveURL(new RegExp(projectPath + "$"));
    await page.getByRole("button", { name: "项目概览", exact: true }).click();
    page.once("dialog", (dialog) => dialog.accept());
    await page.getByRole("button", { name: "归档项目", exact: true }).click();
    await expect(page.getByText("项目已归档 · 仅可查看")).toBeVisible();
    await expect(
      page.getByRole("button", { name: "保存项目资料" }),
    ).toHaveCount(0);
    expect(
      (
        await context.request.patch("/api/v1" + projectPath, {
          data: { name: "归档后应拒绝" },
        })
      ).status(),
    ).toBe(409);
    await page
      .getByRole("link", { name: "返回团队项目", exact: false })
      .click();
    await page.getByLabel("项目状态").selectOption("archived");
    await expect(
      page.getByRole("link", { name: `进入项目 ${projectName} · 第一迭代` }),
    ).toBeVisible();
    const secondResponse = await context.request.post(
      `/api/v1/teams/${team.id}/projects`,
      { data: { name: "团队归档只读验证" } },
    );
    expect(secondResponse.status()).toBe(201);
    const { data: second } = await secondResponse.json();
    expect(
      (await context.request.post(`/api/v1/teams/${team.id}/archive`)).status(),
    ).toBe(204);
    await page.goto(`/projects/${second.id}`);
    await expect(page.getByText("团队已归档 · 仅可查看")).toBeVisible();
    await expect(
      page.getByRole("button", { name: "保存项目资料" }),
    ).toHaveCount(0);
    expect(
      (
        await context.request.post(`/api/v1/teams/${team.id}/projects`, {
          data: { name: "不应创建" },
        })
      ).status(),
    ).toBe(409);
    // Fresh mobile context avoids resizing artifacts from desktop screenshots.
    const mobile = await browser.newContext({
      baseURL,
      viewport: { width: 390, height: 844 },
    });
    try {
      await mobile.addCookies(await context.cookies());
      const mobilePage = await mobile.newPage();
      await mobilePage.goto(`/teams/${team.id}/projects`);
      await expect(
        mobilePage.getByRole("link", {
          name: `进入项目 ${projectName} · 第一迭代`,
        }),
      ).toBeVisible();
      expect(
        await mobilePage.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
      ).toBe(true);
      await mobilePage.screenshot({
        path: testInfo.outputPath("project-list-mobile.png"),
        fullPage: true,
        animations: "disabled",
      });
      await mobilePage.goto(`/projects/${projectID}`);
      await expect(
        mobilePage.getByRole("heading", { name: projectName + " · 第一迭代" }),
      ).toBeVisible();
      expect(
        await mobilePage.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
      ).toBe(true);
      await mobilePage.screenshot({
        path: testInfo.outputPath("project-detail-mobile.png"),
        fullPage: true,
        animations: "disabled",
      });
    } finally {
      await mobile.close();
    }
    expect(errors).toEqual([]);
  } finally {
    await member.close();
    await outside.close();
  }
});
