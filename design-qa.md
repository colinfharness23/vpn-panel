# Portal Design QA

- Visual reference: selected NOVA ice-blue portal mockup
- Reference viewport: 1440 × 1024 CSS pixels
- Verified states: product home, authenticated subscription, plans, clients, password reset, and commercial admin plan editor
- Local screenshots are QA artifacts and are intentionally excluded from Git

## Result

The current portal implementation passed the latest desktop and mobile visual review. No known P0, P1, or P2 visual differences remain against the selected direction.

## Verified areas

- The desktop shell uses a centered wide content frame instead of the earlier narrow, left-aligned layout.
- The home page has complete product information; authenticated subscription details live on a separate route.
- The subscription page follows the selected two-column structure: usage and entitlement on the left, import actions on the right.
- Cards, announcement bar, typography, spacing, borders, ice-blue surfaces, primary actions, and status colors follow the reference hierarchy.
- The 390 × 844 mobile layout has no horizontal overflow and stacks the core cards correctly.
- Navigation, dialogs, feedback, dates, and language names are localized across all 13 supported locales, including RTL layout for Arabic and Persian.
- Password recovery asks for email verification, a new password, and confirmation; it never asks for the forgotten password.
- Plan management exposes create and edit actions, billing prices, visibility, capacity, and 3X-UI inbound binding.
- Client downloads link only to official stores or official GitHub releases.
- Fresh portal and commercial-admin sessions produced no application console errors or warnings.

## Interaction coverage

- Home, subscription, plans, guides, clients, and ticket navigation
- Simplified Chinese and English switching, plus full language labels
- Unauthenticated purchase → authentication dialog
- Email password-reset form with the configured suffix whitelist
- Subscription link copy, QR display, and token rotation
- Desktop and 390 × 844 responsive layouts
- Admin plan list, create dialog, and edit form

## Remaining production validation

Visual QA does not certify payment, email delivery, node synchronization, or server installation. Those require real Alipay sandbox credentials, Gmail SMTP/Turnstile configuration, 3X-UI nodes, and a clean Ubuntu staging host.

Final result: visual and interaction QA passed; external integration and infrastructure acceptance remain required before real-money production launch.

## Admin Login Redesign QA — 2026-07-18

- Source visual truth: `.artifacts/design-qa-login-source.png`
- Desktop implementation: `.artifacts/design-qa-login-desktop.png`
- Mobile implementation: `.artifacts/design-qa-login-mobile.png`
- Full-view comparison: `.artifacts/design-qa-login-comparison.png`
- Focused comparison: `.artifacts/design-qa-login-focus.png`
- Viewports: 1672 × 1272 and 390 × 844 CSS pixels
- State: unauthenticated administrator login, Simplified Chinese, 2FA disabled

### Required fidelity surfaces

- The login page now uses the portal's white and ice-blue palette, deep navy typography, soft blue borders, primary blue action, and restrained card shadow.
- The desktop layout uses a centered two-column card with product context on the left and the login form on the right.
- The mobile layout stacks both sections without horizontal or vertical overflow at 390 × 844.
- The administrator login is fixed to Simplified Chinese and no longer exposes theme or language controls.
- The portal language selector remains unchanged.

### Findings

- No P0, P1, or P2 visual differences remain against the portal design direction.
- The shield mark beside the administrator brand is an acceptable P3 identity detail and does not alter the visual hierarchy.

### Interaction and technical checks

- Username, password, optional 2FA, and login submission behavior remain available.
- Theme control count: 0.
- Language control count: 0.
- Login action count: 1.
- Mobile overflow: none.
- Browser console errors: none.

### Comparison history

- Initial desktop and mobile comparison: no P0, P1, or P2 findings; no corrective iteration required.

final result: passed

## Subscription Settings QA — 2026-07-18

- Source visual: `C:\Users\Administrator\Downloads\e617107567cb858d45550a0f22988fb9.png`
- Implementation screenshot: `C:\Users\ADMINI~1\AppData\Local\Temp\airport-subscription-settings-full.png`
- Side-by-side comparison: `C:\Users\ADMINI~1\AppData\Local\Temp\airport-subscription-settings-comparison.png`
- Browser viewport: 1082 × 1205 CSS pixels
- State: `/panel/settings#subscription`, new `订阅设置` inner tab selected, persisted test value restored

### Required fidelity surfaces

- `订阅设置` is the first inner tab, before the existing `常规`, `信息`, `资料`, `证书`, `Happ`, `Clash / Mihomo`, and `Incy` tabs.
- All nine reference fields are present in the requested top-to-bottom order: user subscription changes, monthly reset mode, offset plan, purchase event, renewal event, change event, subscription path, subscription information output, and protocol name output.
- The existing 3X-UI administrator shell, Ant Design controls, typography, spacing, and page-level save action are retained.
- The existing seven subscription tabs remain mounted; the original `常规` content was opened and verified after the change.

### Findings

- No P0, P1, or P2 visual differences remain.
- The surrounding administrator shell is intentionally retained instead of copying the standalone reference page shell.
- Active switches use the administrator panel's blue design token; field spacing follows the existing site and security settings panes.
- At the 1082-pixel browser width Ant Design moves the last tab into its standard overflow menu; no tab or content is deleted.

### Interaction and technical checks

- Editing enables the existing page-level save action.
- Save, reload, persistence, and restoration were verified in the in-app browser.
- The existing 3X-UI `subPath` setting is reused, avoiding a second conflicting subscription path.
- Disabling user changes removes renewal/upgrade actions in the portal and rejects renewal/upgrade orders server-side.
- Upgrade offset affects integer-fen order pricing; post-provision events can reset traffic; subscription information and protocol-name output are applied by the native subscription service after panel restart.
- Fresh browser console errors and warnings after the updated backend restart: none.
- Frontend tests: 61 files and 818 tests passed.
- Frontend typecheck: passed.
- Production frontend build: passed.
- Go commercial-service, controller, and subscription tests: passed.

### Comparison history

- Initial side-by-side review found the field order and hierarchy matched the source visual while remaining consistent with the existing 3X-UI panel.
- No corrective visual iteration was required; persistence and original-tab preservation were verified separately.

final result: passed

## Security Settings QA — 2026-07-18

- Source visuals: `C:\Users\Administrator\Downloads\a5fea59fb6215f0abe8139c902e9914a.png` and `C:\Users\Administrator\Downloads\8f864e966a0b3da7ae044d972b5ec21b.png`
- Implementation screenshot: `C:\Users\ADMINI~1\AppData\Local\Temp\airport-security-settings-full.png`
- Side-by-side comparison: `C:\Users\ADMINI~1\AppData\Local\Temp\airport-security-settings-comparison.png`
- Browser viewport: 1082 × 1205 CSS pixels
- State: `/panel/settings#security`, `安全设置` selected, password attempt count restored to `5`

### Required fidelity surfaces

- `安全设置` is the first inner tab, before the existing `管理员凭据`, `双重验证`, and `API 令牌` tabs.
- The implementation preserves the existing 3X-UI card, tab, typography, spacing, switch, and form-control language.
- All reference fields are present in the same top-to-bottom order: email verification, Gmail alias restriction, safe mode, backend path, email suffix whitelist, allowed email suffixes, registration CAPTCHA, IP registration limit, password attempt limit, maximum attempts, and lock duration.
- Existing inner tabs remain available and their original content renders correctly.

### Findings

- No P0, P1, or P2 visual differences remain.
- The surrounding 3X-UI administrator shell is intentionally retained instead of copying the standalone reference page shell.
- Active switches use the existing administrator blue token instead of the reference site's dark switch token, keeping the page consistent with the current panel design system.
- `gmail.com` remains the seeded suffix, but the administrator can replace or extend the suffix whitelist; the portal never hard-codes Gmail-only copy.

### Interaction and technical checks

- Editing enables the existing page-level save action.
- Save, reload, persistence, and restoration from `5` to `6` and back to `5` were verified in the browser.
- Email verification, suffix whitelist, Gmail alias rejection, IP registration limits, password lockouts, Turnstile gating, safe-mode host enforcement, and backend-path persistence are backed by server-side policy rather than interface-only switches.
- Fresh browser console errors and warnings: none.
- Frontend tests: 60 files and 817 tests passed.
- Frontend typecheck: passed.
- Production frontend build: passed.
- Go commercial-service and controller tests: passed.

### Comparison history

- Initial full-test run found that an older settings test fixture omitted the backend path value.
- The new pane was made backward-compatible with missing legacy values; the full test suite then passed.
- Final side-by-side review found no remaining P0, P1, or P2 issues.

final result: passed

## Site Settings QA — 2026-07-18

- Source visuals: `C:\Users\Administrator\Downloads\583d2872099a6c7ab23d799d1a4d2610.png` and `C:\Users\Administrator\Downloads\4c99da1509ac53703e2c5d62b3051211.png`
- Implementation screenshot: `C:\Users\ADMINI~1\AppData\Local\Temp\airport-site-settings-full.png`
- Side-by-side comparison: `C:\Users\ADMINI~1\AppData\Local\Temp\airport-site-settings-comparison.png`
- Browser viewport: 1082 × 1205 CSS pixels
- State: `/panel/settings#general`, `站点设置` selected, site name restored to `NOVA`

### Required fidelity surfaces

- `站点设置` is the first inner tab, immediately before the existing `常规` tab.
- The implementation preserves the existing 3X-UI card, tab, typography, spacing, and form-control language.
- All reference fields are present in the same top-to-bottom order: site name, description, URL, HTTPS switch, logo, subscription URLs, terms URL, registration switch, trial plan, currency code, and currency symbol.
- Existing inner tabs remain available: `常规`, `通知`, `证书`, `外部流量`, `日期和时间`, and `LDAP`.

### Findings

- No P0, P1, or P2 visual differences remain.
- The surrounding 3X-UI tab card is intentionally retained instead of copying the reference page shell; this keeps the new content consistent with the existing administrator design system.
- The implementation uses slightly more vertical spacing than the reference, improving readability without changing field hierarchy or order.

### Interaction and technical checks

- Editing enables the existing page-level save action.
- Save, reload, persistence, and restoration to `NOVA` were verified in the browser.
- Registration closure is enforced by the public registration service, not only hidden in the interface.
- Trial-plan selection is backed by active commercial plans and grants the configured plan after successful registration.
- Site URL, logo, subscription base URLs, currency, and currency symbol are exposed through public configuration and consumed by the portal.
- Fresh browser console errors: none.
- Frontend tests: 59 files and 816 tests passed.
- Production frontend build: passed.

### Comparison history

- Initial interaction pass found the save request lacked an explicit JSON content type and returned HTTP 400.
- The request header was corrected; save, reload, and restoration then passed.
- Final side-by-side review found no remaining P0, P1, or P2 issues.

final result: passed

## Invitation & Commission Settings QA — 2026-07-18

- Source visual: `C:\Users\Administrator\Downloads\43c6a42d35ba0ac360624d876e0cfbb3.png`
- Implementation screenshot: `C:\Users\Administrator\AppData\Local\Temp\airport-invitation-settings-full.png`
- Side-by-side comparison: `C:\Users\Administrator\AppData\Local\Temp\airport-invitation-settings-comparison.png`
- Browser viewport: 1067 × 1368 CSS pixels
- State: `/panel/commercial#marketing`, `邀请&佣金设置` selected, default values shown, `开启强制邀请` restored to off after persistence testing

### Required fidelity surfaces

- `邀请&佣金设置` is the first inner tab, before the existing `优惠券`, `礼品卡`, and `邀请佣金` tabs.
- The implementation preserves the existing 3X-UI administrator shell, commercial navigation, Ant Design spacing, typography, switches, and form controls.
- All requested fields are present in the same top-to-bottom order as the reference: forced invitation, commission percentage, invite-code limit, non-expiring codes, first-payment-only commission, automatic confirmation, withdrawal threshold, withdrawal methods, withdrawal closure, and three-level distribution.
- Existing coupon, gift-card, and commission-management tabs remain available and unchanged.

### Findings

- No P0, P1, or P2 visual differences remain.
- The comparison focuses on the requested settings content while intentionally retaining the current administrator chrome and card layout.
- Control values, descriptions, hierarchy, and ordering match the supplied reference; the save action follows the existing panel interaction pattern.

### Interaction and technical checks

- Save, reload, persistence, and restoration of `开启强制邀请` from off to on and back to off were verified in the in-app browser.
- The original `优惠券` tab and its `新建优惠券` action were verified after adding the new first tab.
- Registration enforcement, commission percentage, first-payment-only behavior, three-level distribution, manual settlement, and three-day automatic confirmation are backed by server-side policy.
- Fresh browser console errors and warnings after the final reload: none.

### Comparison history

- The first implementation pass produced deprecation warnings for input adornments and static Ant Design messages.
- Those warnings were removed by using compact controls and the scoped message context.
- The final combined reference/implementation comparison found no remaining P0, P1, or P2 issues.

final result: passed
