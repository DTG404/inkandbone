# Phase 3 Responsive and Accessible UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Preserve Ink & Bone's worn-grimoire identity while making every feature reachable, responsive from 320px upward, keyboard-operable, visibly stateful, and testable for accessibility regressions.

**Architecture:** Establish tested accessible primitives and layout state first, then compose an adaptive three-pane desktop shell, tablet drawers, and four-destination mobile navigation. Migrate CSS by responsibility rather than replacing the entire stylesheet at once.

**Tech Stack:** React 19, TypeScript, CSS, Vitest, Testing Library, Playwright, `@axe-core/playwright`.

## Global Constraints

- Mobile primary destinations are Story, Character, World, and GM.
- Normal text contrast is at least 4.5:1; large text and meaningful controls are at least 3:1.
- Primary touch targets are at least 44 by 44 CSS pixels.
- Every modal supports focus entry, trapping, Escape, and focus restoration.
- Every feature is reachable at 320 CSS pixels without two-dimensional page scrolling.
- The existing visual identity, themes, rulesets, and feature set remain.
- CSS migration is incremental and covered by screenshots/interaction tests.

---

### Task 1: Repair the frontend test baseline

**Files:**
- Modify: `web/src/test-setup.ts`
- Modify: `web/src/CharacterSheetPanel.test.tsx`
- Modify: `web/src/MapPanel.test.tsx`
- Modify: `web/src/MapPanel.tsx`

**Interfaces:**
- Produces: deterministic `ResizeObserver` test implementation and complete map endpoint mocks.

- [ ] **Step 1: Confirm the three existing failures**

Run: `cd web && npm test -- --run`

Expected baseline: 119 passed and 3 failed, including computed-field timing and map token/zone setup.

- [ ] **Step 2: Make the computed-field test await observable state**

Replace the immediate assertion with:

```tsx
await waitFor(() => expect(screen.getByLabelText('Proficiency Bonus')).toHaveValue(2))
```

Do not add production delays or test-only component branches.

- [ ] **Step 3: Add a standards-shaped ResizeObserver test double**

```ts
class TestResizeObserver implements ResizeObserver {
  observe(): void {}
  unobserve(): void {}
  disconnect(): void {}
}
globalThis.ResizeObserver = TestResizeObserver
```

- [ ] **Step 4: Extend MapPanel mocks for tokens and zones**

Return empty arrays for `/api/maps/{id}/tokens` and `/api/maps/{id}/zones`; assert the component actually issues both reads. Remove the stale unused ESLint suppression from `MapPanel.tsx`.

- [ ] **Step 5: Verify the clean baseline and commit**

Run: `cd web && npm test -- --run && npm run lint`

Expected: 122/122 tests pass, 0 errors, 0 warnings.

```bash
git add web/src/test-setup.ts web/src/CharacterSheetPanel.test.tsx web/src/MapPanel.test.tsx web/src/MapPanel.tsx
git commit -m "test: repair frontend regression baseline"
```

### Task 2: Build accessible UI primitives test-first

**Files:**
- Create: `web/src/ui/Dialog.tsx`
- Create: `web/src/ui/Dialog.test.tsx`
- Create: `web/src/ui/Drawer.tsx`
- Create: `web/src/ui/Tabs.tsx`
- Create: `web/src/ui/IconButton.tsx`
- Create: `web/src/ui/StatusRegion.tsx`
- Create: `web/src/ui/ui.test.tsx`
- Create: `web/src/styles/primitives.css`
- Modify: `web/src/App.tsx`

**Interfaces:**
- Produces: `Dialog`, `Drawer`, `TabList`, `Tab`, `TabPanel`, `IconButton`, and `StatusRegion` components.

- [ ] **Step 1: Write failing semantic and keyboard tests**

Test `role=dialog`, `aria-modal`, labelled title, initial focus, Tab/Shift+Tab wrap, Escape close, opener focus restoration, overlay close, selected tab/controlled panel relationships, ArrowLeft/ArrowRight/Home/End tab movement, required icon-button label, and polite/assertive status regions.

```tsx
it('restores focus to the opener after Escape', async () => {
  const user = userEvent.setup()
  render(<DialogHarness />)
  const opener = screen.getByRole('button', { name: 'Open' })
  await user.click(opener)
  await user.keyboard('{Escape}')
  expect(opener).toHaveFocus()
})
```

- [ ] **Step 2: Run and verify RED**

Run: `cd web && npm test -- --run ui`

Expected: module-not-found failures for the primitives.

- [ ] **Step 3: Implement Dialog and Drawer focus management**

Use `useId`, refs, a document keydown listener only while open, a query for native focusable controls, previous-active-element restoration, body scroll locking with cleanup, and a rendered backdrop. Drawer composes Dialog semantics at narrow widths and may be non-modal only when explicitly configured.

- [ ] **Step 4: Implement tabs, icon buttons, and live status**

Tabs use real buttons with roving `tabIndex`; `TabPanel` uses `hidden` and `aria-labelledby`; `IconButton` requires `label: string`; `StatusRegion` renders a stable `role=status` or `role=alert` node so announcements are not lost during remount.

- [ ] **Step 5: Add shared focus and target styles**

All interactive primitives receive a visible two-pixel focus ring with offset and a minimum 44px touch target under coarse-pointer media queries.

- [ ] **Step 6: Verify and commit**

Run: `cd web && npm test -- --run ui && npm run lint`

```bash
git add web/src/ui web/src/styles/primitives.css web/src/App.tsx
git commit -m "feat: add accessible interaction primitives"
```

### Task 3: Add persistent, resettable adaptive layout state

**Files:**
- Create: `web/src/layout/useWorkspaceLayout.ts`
- Create: `web/src/layout/useWorkspaceLayout.test.tsx`
- Create: `web/src/layout/WorkspaceShell.tsx`
- Create: `web/src/layout/WorkspaceShell.test.tsx`
- Create: `web/src/styles/layout.css`
- Modify: `web/src/App.tsx`
- Modify: `web/src/SessionView.tsx`

**Interfaces:**
- Produces: `WorkspaceLayout{leftWidth,rightWidth,leftCollapsed,rightCollapsed}`, `resetLayout`, and desktop/tablet/mobile shell slots.

- [ ] **Step 1: Write failing layout-state tests**

Assert defaults, clamping, persistence under key `inkandbone.workspace.v1`, corrupt-storage recovery, reset behavior, pointer and keyboard resizing, and a restore control that remains visible after collapse.

```ts
const DEFAULT_LAYOUT = { leftWidth: 420, rightWidth: 320, leftCollapsed: false, rightCollapsed: false }
const LIMITS = { left: [280, 620], right: [260, 520] } as const
```

- [ ] **Step 2: Run and verify RED**

Run: `cd web && npm test -- --run WorkspaceLayout`

- [ ] **Step 3: Implement the versioned layout hook**

Parse storage defensively, clamp widths on load and change, persist only validated values, and remove the storage key on reset. Expose semantic resize separators with arrow-key increments of 16px and Shift+Arrow increments of 48px.

- [ ] **Step 4: Implement WorkspaceShell**

Use CSS custom properties `--left-panel-width` and `--right-panel-width`; render named `header`, `left`, `story`, `right`, and `mobileNav` slots; use `100dvh`; keep restore buttons outside collapsed panels.

- [ ] **Step 5: Compose current SessionView content without changing feature behavior**

Move only shell ownership from the current `.grimoire-body` structure. Keep existing callbacks, API calls, and panel contents unchanged in this commit.

- [ ] **Step 6: Verify and commit**

Run: `cd web && npm test -- --run Workspace && npm run build`

```bash
git add web/src/layout web/src/styles/layout.css web/src/App.tsx web/src/SessionView.tsx
git commit -m "feat: add adaptive resizable workspace shell"
```

### Task 4: Replace clipped tabs with grouped responsive navigation

**Files:**
- Create: `web/src/navigation/panelRegistry.tsx`
- Create: `web/src/navigation/WorkspaceNavigation.tsx`
- Create: `web/src/navigation/WorkspaceNavigation.test.tsx`
- Create: `web/src/styles/navigation.css`
- Modify: `web/src/SessionView.tsx`

**Interfaces:**
- Produces: `PanelID`, `PanelGroup`, registry entries `{id,label,group,render}`, and mobile destinations `story`, `character`, `world`, `gm`.

- [ ] **Step 1: Write failing reachability tests for every current right-panel feature**

Build the expected list from the current 15 tabs and assert each label is visible or reachable through its Play, World, or GM group by keyboard. At 390px assert Story, Character, World, and GM are direct bottom-navigation buttons with selected state.

- [ ] **Step 2: Run and verify RED**

Run: `cd web && npm test -- --run WorkspaceNavigation`

- [ ] **Step 3: Define the registry and grouping**

Use the 15 stable IDs already present in `rightTab`. Group handouts, decks, and oracle under Play; compendium, notes, journal, NPCs, relationships, factions, and calendar under World; objectives, NPC stat blocks, adventures, secrets, and GM tools under GM. Character and Story remain direct workspace destinations rather than duplicated right-panel entries.

- [ ] **Step 4: Render desktop grouped tabs and mobile destinations**

Desktop uses the accessible Tabs primitives with visible group headings and an overflow-safe vertical list. Tablet invokes drawers. Mobile bottom navigation switches workspace destination while secondary panel selection remains inside World or GM.

- [ ] **Step 5: Verify and commit**

Run: `cd web && npm test -- --run 'WorkspaceNavigation|SessionView' && npm run build`

```bash
git add web/src/navigation web/src/styles/navigation.css web/src/SessionView.tsx
git commit -m "feat: group and expose workspace navigation"
```

### Task 5: Correct theme contrast, typography, motion, and reflow

**Files:**
- Create: `web/src/styles/tokens.css`
- Create: `web/src/styles/responsive.css`
- Modify: `web/src/App.css`
- Delete: `web/src/index.css` after verifying it is unreferenced.
- Create: `web/src/styles/contrast.test.ts`

**Interfaces:**
- Produces: canonical dark/light tokens, breakpoint behavior at 1200px, 900px, and 600px, reduced-motion overrides.

- [ ] **Step 1: Write failing contrast-token tests**

Implement a small test-only WCAG relative-luminance helper and assert every foreground/background token pair named in a table meets 4.5 or 3.0 as appropriate. Include muted labels, active tabs, borders used as state indicators, errors, health, and parchment theme.

- [ ] **Step 2: Run and verify RED**

Run: `cd web && npm test -- --run contrast.test.ts`

Expected: current `--gold-dim` combinations fail.

- [ ] **Step 3: Establish accessible tokens and base sizing**

Import `tokens.css` before existing rules. Raise body text to at least 14px, ordinary labels to at least 12px, preserve serif headings, and replace low-contrast semantic uses of `--gold-dim`. Decorative borders may remain lower contrast when they do not communicate state.

- [ ] **Step 4: Add responsive and safe-area rules**

Use `height: 100dvh`, `env(safe-area-inset-*)`, single-axis scrolling per region, tablet drawers below 900px, mobile workspace below 600px, and no fixed 520px/260px widths. Verify long labels wrap or scroll without clipping.

- [ ] **Step 5: Add reduced-motion behavior**

Under `@media (prefers-reduced-motion: reduce)`, remove infinite pulses, particles, flashing conditions, smooth scrolling, and nonessential transitions while retaining immediate state changes.

- [ ] **Step 6: Remove stale scaffold CSS and verify**

Run: `rg -n "index.css" web/src web/index.html` and delete the file only when no import exists.

Run: `cd web && npm test -- --run && npm run lint && npm run build`.

- [ ] **Step 7: Commit**

```bash
git add web/src/App.css web/src/styles web/src/index.css
git commit -m "fix: improve theme contrast motion and reflow"
```

### Task 6: Migrate modals and interactive rows to accessible primitives

**Files:**
- Modify: `web/src/ManagePanel.tsx`
- Modify: `web/src/GMScreenPanel.tsx`
- Modify: `web/src/SessionView.tsx`
- Modify panels found by `rg -n '<div[^>]+onClick=' web/src`.
- Add focused tests beside each affected component.

**Interfaces:**
- Consumes: Phase 3 accessible primitives.

- [ ] **Step 1: Add failing keyboard tests for current overlays and clickable rows**

Cover Manage, GM Screen, handout, history, talents, and map-pin dialogs; campaign-selection rows; Secrets, Factions, NPC stat, and Adventures rows; icon-only close/add/delete buttons.

- [ ] **Step 2: Run and verify RED**

Run: `cd web && npm test -- --run 'Manage|GMScreen|Dialog|Keyboard'`

- [ ] **Step 3: Replace overlays with Dialog/Drawer and rows with buttons**

Use native `<button type="button">` for actions. When a row also contains secondary buttons, make its primary action a dedicated button rather than nesting buttons. Supply meaningful `aria-label` text for every icon-only action.

- [ ] **Step 4: Verify static searches**

Run: `rg -n '<div[^>]+onClick=|outline:\s*none' web/src`.

Expected: no clickable div remains; every removed outline has a matching `:focus-visible` rule.

- [ ] **Step 5: Verify and commit**

Run: `cd web && npm test -- --run && npm run lint`

```bash
git add web/src
git commit -m "fix: make dialogs and actions keyboard accessible"
```

### Task 7: Add visible async, connection, and automation-health states

**Files:**
- Create: `web/src/ui/ToastProvider.tsx`
- Create: `web/src/ui/ToastProvider.test.tsx`
- Modify: `web/src/App.tsx`
- Modify: `web/src/AutomationSettingsPanel.tsx`
- Modify: `web/src/AutomationSettingsPanel.test.tsx`
- Modify components containing `.catch(() =>` or `console.error` for user-triggered actions.

**Interfaces:**
- Produces: `useToast`, persistent connection banner, retry actions, automation statuses from Phase 2.

- [ ] **Step 1: Write failing user-feedback tests**

Assert sending, map generation, recap, campaign management, and GM tool failures announce an error with a safe retry action; offline/reconnecting states remain visible; successful destructive actions announce completion; automation cooling-down state shows next-probe guidance.

- [ ] **Step 2: Run and verify RED**

Run: `cd web && npm test -- --run 'Toast|AutomationSettings|App'`

- [ ] **Step 3: Implement stable live regions and bounded toast queue**

Keep at most five messages, default informational expiry to five seconds, retain errors until dismissed, pause expiry while hovered/focused, and never put raw provider/database errors in user-visible text.

- [ ] **Step 4: Replace silent user-action failures**

Keep console logging only as supplementary development evidence. Restore user input on failed message send and expose explicit retry for idempotent operations.

Add a first-run empty state when no active campaign exists. It explains the three required steps—create a campaign, create/select a character, create/select a session—and opens the relevant Manage tab from a real button. Show AI backend availability without implying cloud providers are local.

- [ ] **Step 5: Verify and commit**

Run: `cd web && npm test -- --run && npm run lint`

```bash
git add web/src
git commit -m "feat: expose recoverable ui and automation states"
```

### Task 8: Add responsive and accessibility browser gates

**Files:**
- Modify: `e2e/package.json`
- Modify: `e2e/package-lock.json`
- Create: `e2e/tests/accessibility.spec.ts`
- Create: `e2e/tests/responsive.spec.ts`
- Create: `docs/accessibility-release-checklist.md`

**Interfaces:**
- Adds: exact dev dependency `@axe-core/playwright@4.12.1`.

- [ ] **Step 1: Install the pinned accessibility test helper**

Run: `cd e2e && npm install --save-dev --save-exact @axe-core/playwright@4.12.1`

Expected: only `e2e/package.json` and `e2e/package-lock.json` change.

- [ ] **Step 2: Write browser tests before changing any remaining UI defects**

At 1440×900, 1024×768, 768×1024, 390×844, and 320×568, assert primary destinations and story input are reachable, no document horizontal overflow exists, dialogs stay within viewport, and axe reports no serious/critical violations. Add keyboard-only campaign/session selection, first-run onboarding, and GM-dialog flows. Capture deterministic dark/light desktop and mobile screenshots with animation disabled so later visual regressions are reviewable.

- [ ] **Step 3: Run tests and verify any residual defects fail**

Run: `cd e2e && npm test -- accessibility.spec.ts responsive.spec.ts`

- [ ] **Step 4: Fix only defects demonstrated by the new tests**

Apply component-local semantic or CSS corrections, rerunning the focused spec after each correction.

- [ ] **Step 5: Write the manual release checklist**

Include keyboard-only navigation, NVDA/VoiceOver spot checks, 200 percent zoom, text spacing, both themes, reduced motion, touch targets, focus order, and announcement behavior with pass/fail/date fields.

- [ ] **Step 6: Run complete Phase 3 verification**

Run:

```bash
cd web && npm test -- --run && npm run lint && npm run build
cd ../e2e && npm test
```

Expected: all unit, accessibility, responsive, and maintained browser tests pass without unhandled errors.

- [ ] **Step 7: Commit**

```bash
git add e2e docs/accessibility-release-checklist.md web/src
git commit -m "test: gate responsive accessible workflows"
```
