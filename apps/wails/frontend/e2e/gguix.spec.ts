import { test, expect, type Page } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";
import { PREFS_DIR } from "../playwright.config";
import oracle from "./oracle.json" with { type: "json" };

const items = oracle.firstItems;

test.beforeEach(async ({ page }) => {
  fs.rmSync(path.join(PREFS_DIR, "prefs.json"), { force: true });
  await page.goto("/");
  await expect(page.getByTestId("row").first()).toBeVisible();
});

const row = (page: Page, i: number) => page.locator(`[data-testid=row][data-index="${i}"]`);

test("T1 app starts and loads", async ({ page }) => {
  for (let i = 0; i < 3; i++) {
    await expect(row(page, i).locator("[data-col=id]")).toHaveText(String(items[i].id));
    await expect(row(page, i).locator("[data-col=sku]")).toHaveText(items[i].sku);
  }
});

test("T2 sort by name", async ({ page }) => {
  await page.getByTestId("th-name").click();
  await expect(page.getByTestId("th-name")).toContainText("▲");
  await expect(row(page, 0).locator("[data-col=name]")).toHaveText(oracle.nameFirst);
});

test("T3 search updates the count", async ({ page }) => {
  await page.getByPlaceholder("Search name or SKU").fill("steel");
  await expect(page.getByTestId("status-left")).toContainText(`${oracle.steelTotal} items`);
});

test("T4 selection fills the form", async ({ page }) => {
  const it = items[2];
  await row(page, 2).click();
  await expect(page.getByTestId("f-id")).toHaveText(String(it.id));
  await expect(page.getByTestId("f-sku")).toHaveText(it.sku);
  await expect(page.locator("#f-name")).toHaveValue(it.name);
  await expect(page.getByTestId("f-category")).toHaveText(it.category);
  await expect(page.locator("#f-price")).toHaveValue(it.priceText);
  await expect(page.locator("#f-qty")).toHaveValue(String(it.quantity));
  await expect(page.locator(`#st-${it.status}`)).toBeChecked();
  await expect(page.getByTestId("f-rating-text")).toHaveText(it.ratingText);
  if (it.featured) await expect(page.locator("#f-featured")).toBeChecked();
  else await expect(page.locator("#f-featured")).not.toBeChecked();
  await expect(page.getByTestId("f-updated")).toHaveText(it.updatedText);
  await expect(page.locator("#f-notes")).toHaveValue(it.notes);
});

test("T5 validation blocks save", async ({ page }) => {
  await row(page, 0).click();
  await page.locator("#f-name").fill("ab");
  await page.getByRole("button", { name: "Save" }).click();
  await expect(page.getByTestId("err-name")).toHaveText("Name must be 3 to 80 characters.");
  await expect(page.getByText("Unsaved changes")).toBeVisible();
  await expect(row(page, 0).locator("[data-col=name]")).toHaveText(items[0].name);
});

test("T6 save updates the row", async ({ page }) => {
  await row(page, 4).click();
  await page.locator("#f-name").fill("Renamed Test Item");
  await page.getByRole("button", { name: "Save" }).click();
  await expect(row(page, 4).locator("[data-col=name]")).toHaveText("Renamed Test Item");
  await expect(page.getByText("Unsaved changes")).toHaveCount(0);
});

test("T7 theme and prefs", async ({ page }) => {
  await page.getByTestId("theme").click();
  await page.getByRole("option", { name: "Dark" }).click();
  await expect(page.locator("html")).toHaveClass(/dark/);
  await expect.poll(() => fs.existsSync(path.join(PREFS_DIR, "prefs.json")) && fs.readFileSync(path.join(PREFS_DIR, "prefs.json"), "utf8"), { timeout: 5000 }).toContain('"theme":"dark"');
});

test("T8 export cancel", async ({ page }) => {
  const btn = page.getByRole("button", { name: "Export CSV" });
  await btn.click();
  await expect(page.getByTestId("export-progress")).toBeVisible();
  await expect(btn).toBeDisabled();
  await page.waitForTimeout(500);
  await page.getByRole("button", { name: "Cancel" }).click();
  await expect(page.getByText("Export cancelled")).toBeVisible();
  await expect(btn).toBeEnabled();
  await expect(page.getByTestId("export-progress")).toHaveCount(0);
});

test("T9 dirty guard", async ({ page }) => {
  await row(page, 0).click();
  await page.locator("#f-name").fill("Changed Name");
  await row(page, 1).click();
  await expect(page.getByTestId("confirm-dialog")).toContainText("Discard unsaved changes?");
  await page.getByRole("button", { name: "Cancel" }).click();
  await expect(page.getByTestId("f-id")).toHaveText("1");
  await expect(page.locator("#f-name")).toHaveValue("Changed Name");
});

for (const dark of [false, true]) {
  test(`T10 custom cells (${dark ? "dark" : "light"})`, async ({ page }) => {
    await page.emulateMedia({ colorScheme: dark ? "dark" : "light" });
    await page.reload();
    const it = items[0];
    const pal = dark ? oracle.palette.dark : oracle.palette.light;
    const r = row(page, 0);
    await expect(r.getByTestId("stock-bar")).toHaveAttribute("data-fraction", String(it.stockFraction));
    await expect(r.getByTestId("stock-fill")).toHaveAttribute("width", String(80 * it.stockFraction));
    await expect(r.getByTestId("stock-bar")).toHaveAttribute("data-fill", it.stockLow ? pal.stockFillLow : pal.stockFill);
    const badge = r.getByTestId("status-badge");
    const b = it.status === "Backordered" ? pal.backordered : it.status === "Discontinued" ? pal.discontinued : pal.active;
    await expect(badge).toHaveText(it.status);
    await expect(badge).toHaveAttribute("data-bg", b.background);
    await expect(badge).toHaveAttribute("data-fg", b.text);
    const stars = r.getByTestId("stars");
    await expect(stars).toHaveAttribute("data-full", String(it.starsFull));
    await expect(stars).toHaveAttribute("data-half", String(it.starsHalf));
    await expect(stars).toHaveAttribute("data-empty", String(it.starsEmpty));
    await expect(stars).toContainText(it.ratingText);
  });
}

test("T11 cards view", async ({ page }) => {
  await page.getByRole("button", { name: "Cards", exact: true }).click();
  const c1 = page.locator(`[data-testid=card][data-id="${items[0].id}"]`);
  await expect(c1).toContainText(items[0].name);
  await expect(c1).toContainText(items[0].priceText);
  await expect(c1.getByTestId("status-badge")).toHaveText(items[0].status);
  await page.locator(`[data-testid=card][data-id="${items[1].id}"]`).click();
  await expect(page.getByTestId("f-id")).toHaveText(String(items[1].id));
  await expect(page.locator("#f-name")).toHaveValue(items[1].name);
});

test("T12 host info", async ({ page }) => {
  await expect(page.getByTestId("identity")).toHaveText(`${oracle.serverUser}@${oracle.serverHost}`);
  await page.getByRole("button", { name: "Host", exact: true }).click();
  const d = page.getByTestId("host-dialog");
  await expect(d.getByTestId("host-Source")).toHaveText("server-process");
  await expect(d.getByTestId("host-Username")).toHaveText(oracle.serverUser);
  await expect(d.getByRole("button", { name: "Reveal Config Folder" })).toBeDisabled();
});

test("T13 split divider drags", async ({ page }) => {
  const handle = page.locator("[data-slot=resizable-handle]");
  const pane = page.locator("[data-pane]").first();
  const before = (await pane.boundingBox())!.width;
  const hb = (await handle.boundingBox())!;
  await page.mouse.move(hb.x + hb.width / 2, hb.y + hb.height / 2);
  await page.mouse.down();
  await page.mouse.move(hb.x - 150, hb.y + hb.height / 2, { steps: 8 });
  await page.mouse.up();
  expect((await pane.boundingBox())!.width).toBeLessThan(before - 100);
});
