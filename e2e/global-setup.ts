// globalSetup runs AFTER the webServer starts in Playwright ≥1.25.
// We cannot delete the DB here because the server already has it open.
// Cleanup happens in global-teardown.ts after the server stops.
export const DB_PATH = '/tmp/ttrpg-smoke-test.db'

export default function globalSetup() {
  // intentionally empty — cleanup is handled post-run by globalTeardown
}
