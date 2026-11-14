import { defineConfig } from "@playwright/test";
import os from "node:os";
import path from "node:path";

export const PREFS_DIR = path.join(os.tmpdir(), "gguix-mygo-e2e-prefs");
const PORT = 8191;

// MyGo has no browser mode, so the tests drive a test-only HTTP harness
// (../main_e2e.go, tag e2e) that serves the built frontend and the real Go
// service (core included) in Google Chrome, with window.mygo replaced by
// e2e/shim.js. The BRIEF section 8 settings: 1000 rows, no latency, exports
// throttled to 200 rows per second.
export default defineConfig({
  testDir: "e2e",
  timeout: 30_000,
  workers: 1,
  reporter: [["list"], ["json", { outputFile: "e2e/results.json" }]],
  use: { baseURL: `http://127.0.0.1:${PORT}`, channel: "chrome", viewport: { width: 1600, height: 900 }, colorScheme: "light" },
  webServer: {
    command: "bin/gguix-mygo-e2e",
    url: `http://127.0.0.1:${PORT}/health`,
    reuseExistingServer: false,
    cwd: "..",
    env: {
      GGUIX_ROWS: "1000",
      GGUIX_LATENCY_MS: "0",
      GGUIX_EXPORT_RPS: "200",
      GGUIX_PREFS_DIR: PREFS_DIR,
      GGUIX_E2E_PORT: String(PORT),
    },
  },
});
