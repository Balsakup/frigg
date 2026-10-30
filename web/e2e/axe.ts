import AxeBuilder from "@axe-core/playwright";
import { expect, type Page } from "@playwright/test";

// Seuil fixé par la constitution : aucune violation `serious` ou `critical`.
const BLOCKING = new Set(["serious", "critical"]);

export async function expectNoBlockingA11yViolations(page: Page) {
  const { violations } = await new AxeBuilder({ page })
    .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"])
    .analyze();
  const blocking = violations
    .filter((v) => v.impact && BLOCKING.has(v.impact))
    .map((v) => ({ id: v.id, impact: v.impact, nodes: v.nodes.length, help: v.help }));
  expect(blocking).toEqual([]);
}
