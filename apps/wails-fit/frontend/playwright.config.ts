import { defineConfig } from "@playwright/test";
import os from "node:os";
import path from "node:path";

export const PREFS_DIR = path.join(os.tmpdir(), "gguix-wails-e2e-prefs");
const PORT = 8191;

// The tests drive the real server-mode build (Go core included) in Google
// Chrome, with the BRIEF section 8 settings: 1000 rows, no latency, exports
// throttled to 200 rows per second.
export default defineConfig({
  testDir: "e2e",
  timeout: 30_000,
  workers: 1,
  reporter: [["list"], ["json", { outputFile: "e2e/results.json" }]],
  use: { baseURL: `http://localhost:${PORT}`, channel: "chrome", viewport: { width: 1600, height: 900 }, colorScheme: "light" },
  webServer: {
    command: "../bin/gguix-wails-server",
    url: `http://localhost:${PORT}/health`,
    reuseExistingServer: false,
    env: {
      GGUIX_ROWS: "1000",
      GGUIX_LATENCY_MS: "0",
      GGUIX_EXPORT_RPS: "200",
      GGUIX_PREFS_DIR: PREFS_DIR,
      WAILS_SERVER_PORT: String(PORT),
    },
  },
});
