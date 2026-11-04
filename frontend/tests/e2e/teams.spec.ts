import { test, expect } from "@playwright/test";
import { randomUUID } from "node:crypto";

test("团队创建、真实邀请、角色与所有权、退出和归档", async ({
  page,
  context,
  browser,
  baseURL,
}, testInfo) => {
  const password = `Team-${randomUUID()}`,
    name = `协作平台研发组-${randomUUID().slice(0, 8)}`;
  const member = await browser.newContext({ baseURL }),
    outsider = await browser.newContext({ baseURL });
  const ownerEmail = `owner-${randomUUID()}@example.test`,
    memberEmail = `member-${randomUUID()}@example.test`,
    outsiderEmail = `outside-${randomUUID()}@example.test`;
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));
  try {
    for (const [ctx, email, nickname] of [
      [context, ownerEmail, "团队所有者"],
      [member, memberEmail, "伙伴同学"],
      [outsider, outsiderEmail, "团队外成员"],
    ] as const) {
      expect(
        (
          await ctx.request.post("/api/v1/auth/register", {
            data: { email, nickname, password },
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
    await page.goto("/teams");
    await page.getByRole("button", { name: "创建团队", exact: true }).click();
    await page.getByLabel("团队名称", { exact: true }).fill(name);
    await page
      .getByLabel("团队简介", { exact: true })
      .fill("一起完成 Go 与 Vue 团队协作平台，保持目标清晰、分工明确。");
    await page.getByRole("button", { name: "确认创建" }).click();
    await expect(page).toHaveURL(/\/teams\/\d+$/);
    const teamPath = new URL(page.url()).pathname;
    await expect(
      page.getByRole("heading", { name, exact: true }),
    ).toBeVisible();
    const memberPage = await member.newPage();
    memberPage.on("pageerror", (e) => errors.push(e.message));
    await memberPage.goto(teamPath);
    await expect(memberPage.getByRole("alert")).toContainText("没有访问权限");
    await page.getByRole("button", { name: "邀请码", exact: true }).click();
    await page.getByLabel("最多使用次数").fill("2");
    await page.getByRole("button", { name: "创建邀请码", exact: true }).click();
    const inviteInput = page.getByLabel("新邀请码（离开页面后不再显示）");
    await expect(inviteInput).toBeVisible();
    const code = await inviteInput.inputValue();
    expect(code).toMatch(/^[0-9a-f]{64}$/);
    await page.screenshot({
      path: testInfo.outputPath("team-invites.png"),
      fullPage: true,
      animations: "disabled",
      mask: [inviteInput],
    });
    await memberPage.goto("/teams");
    await memberPage.getByRole("button", { name: "邀请码加入" }).click();
    await memberPage.getByLabel("邀请码", { exact: true }).fill(code);
    await memberPage.getByRole("button", { name: "确认加入" }).click();
    await expect(memberPage).toHaveURL(new RegExp(teamPath + "$"));
    await expect(
      memberPage.getByText("我的角色：成员", { exact: false }),
    ).toBeVisible();
    await expect(
      memberPage.getByRole("button", { name: "邀请码", exact: true }),
    ).toHaveCount(0);
    await expect(
      memberPage.getByRole("button", { name: "保存团队资料" }),
    ).toHaveCount(0);
    await page.getByRole("button", { name: "成员", exact: true }).click();
    const row = page.locator("tbody tr").filter({ hasText: memberEmail });
    await expect(row).toBeVisible();
    await row.getByRole("button", { name: "设为管理员" }).click();
    await expect(row.getByText("管理员", { exact: true })).toBeVisible();
    await page.screenshot({
      path: testInfo.outputPath("team-members.png"),
      fullPage: true,
      animations: "disabled",
    });
    await page.setViewportSize({width:390,height:844});
    expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
    await page.screenshot({path:testInfo.outputPath("team-members-mobile.png"),fullPage:true,animations:"disabled"});
    await page.setViewportSize({width:1440,height:1000});
    await memberPage.reload();
    await expect(
      memberPage.getByText("我的角色：管理员", { exact: false }),
    ).toBeVisible();
    await memberPage.getByRole("button", { name: "成员", exact: true }).click();
    await expect(
      memberPage.getByRole("button", { name: "转让所有权" }),
    ).toHaveCount(0);
    page.once("dialog", (dialog) => dialog.accept());
    await row.getByRole("button", { name: "转让所有权" }).click();
    await expect(
      page.getByText("我的角色：管理员", { exact: false }),
    ).toBeVisible();
    await page.getByRole("button", { name: "概览", exact: true }).click();
    await expect(page.getByRole("button", { name: "归档团队" })).toHaveCount(0);
    page.once("dialog", (dialog) => dialog.accept());
    await page.getByRole("button", { name: "退出团队" }).click();
    await expect(page).toHaveURL(/\/teams$/);
    expect((await context.request.get("/api/v1" + teamPath)).status()).toBe(
      404,
    );
    await memberPage.reload();
    await expect(
      memberPage.getByText("我的角色：所有者", { exact: false }),
    ).toBeVisible();
    await memberPage
      .getByRole("button", { name: "团队动态", exact: true })
      .click();
    await expect(memberPage.locator(".activity-list")).toContainText(
      "转让了团队所有权",
    );
    await expect(memberPage.locator(".activity-list")).toContainText(
      "退出了团队",
    );
    await memberPage
      .getByRole("button", { name: "邀请码", exact: true })
      .click();
    await memberPage
      .getByRole("button", { name: "创建邀请码", exact: true })
      .click();
    const newInput = memberPage.getByLabel("新邀请码（离开页面后不再显示）");
    await expect(newInput).toBeVisible();
    const revokedCode = await newInput.inputValue();
    const firstRow = memberPage.locator("tbody tr").first();
    memberPage.once("dialog", (dialog) => dialog.accept());
    await firstRow.getByRole("button", { name: "撤销", exact: true }).click();
    await expect(firstRow).toContainText("已撤销");
    await expect(newInput).toHaveCount(0);
    const outsidePage = await outsider.newPage();
    await outsidePage.goto("/teams");
    await outsidePage.getByRole("button", { name: "邀请码加入" }).click();
    await outsidePage.getByLabel("邀请码", { exact: true }).fill(revokedCode);
    await outsidePage.getByRole("button", { name: "确认加入" }).click();
    await expect(outsidePage.getByRole("alert")).toContainText("邀请码无效");
    await memberPage.getByRole("button", { name: "概览", exact: true }).click();
    memberPage.once("dialog", (dialog) => dialog.accept());
    await memberPage.getByRole("button", { name: "归档团队" }).click();
    await expect(memberPage.getByText("已归档 · 仅可查看")).toBeVisible();
    await expect(
      memberPage.getByRole("button", { name: "保存团队资料" }),
    ).toHaveCount(0);
    await memberPage
      .getByRole("button", { name: "邀请码", exact: true })
      .click();
    await expect(
      memberPage.getByRole("button", { name: "创建邀请码", exact: true }),
    ).toHaveCount(0);
    await outsidePage.getByLabel("邀请码", { exact: true }).fill(code);
    await outsidePage.getByRole("button", { name: "确认加入" }).click();
    await expect(outsidePage.getByRole("alert")).toContainText("团队已归档");
    await memberPage.goto("/teams");
    await memberPage.getByLabel("团队状态").selectOption("archived");
    await expect(
      memberPage.getByRole("link", { name: `进入团队 ${name}` }),
    ).toBeVisible();
    await memberPage.setViewportSize({ width: 390, height: 844 });
    expect(
      await memberPage.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    await memberPage.screenshot({
      path: testInfo.outputPath("team-list-mobile.png"),
      fullPage: true,
      animations: "disabled",
    });
    expect(errors).toEqual([]);
  } finally {
    await member.close();
    await outsider.close();
  }
});
