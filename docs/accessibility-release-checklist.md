# Accessibility Release Checklist

Use this checklist for every release that changes the workspace shell, navigation, dialogs, forms, notifications, or theme tokens. Record `Pass`, `Fail`, or `N/A` and the date for every row; link a defect when a check fails.

| Check | Result | Date | Notes or defect |
|---|---|---|---|
| Keyboard-only: reach all primary destinations and story controls without a pointer |  |  |  |
| Keyboard-only: create/select a campaign, character, and session |  |  |  |
| Keyboard-only: open, traverse, close, and restore focus for every dialog |  |  |  |
| Focus order follows the visual and task order with no traps outside open dialogs |  |  |  |
| NVDA + Firefox or Chrome: landmarks, headings, controls, validation, and updates are announced clearly |  |  |  |
| VoiceOver + Safari: landmarks, headings, controls, validation, and updates are announced clearly |  |  |  |
| Status announcements: connection, success, error, retry, and destructive-action results are timely and not duplicated |  |  |  |
| 200% browser zoom: all content and controls remain reachable without two-dimensional scrolling |  |  |  |
| Text spacing: WCAG text-spacing overrides do not clip, overlap, or hide content |  |  |  |
| Worn Grimoire theme: text, focus, selection, status, and meaningful borders remain perceivable |  |  |  |
| Parchment theme: text, focus, selection, status, and meaningful borders remain perceivable |  |  |  |
| Reduced motion: animations and smooth scrolling are removed without losing state or meaning |  |  |  |
| Touch targets: primary and repeated controls are at least 44 by 44 CSS pixels on a coarse pointer |  |  |  |
| Mobile 320×568 and 390×844: safe-area padding, dialogs, story input, and bottom navigation stay in the viewport |  |  |  |
| Tablet 768×1024: Story is unobstructed by default; Character/Tools open one labelled drawer at a time and restore opener focus |  |  |  |
| Desktop 1024×768/1440×900: resizable panels remain reachable without document horizontal overflow |  |  |  |

## Release record

- Release or commit:
- Reviewer:
- Assistive technology and versions:
- Browser and operating system versions:
- Overall result: Pass / Fail
- Follow-up defects:
