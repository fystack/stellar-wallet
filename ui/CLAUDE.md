# UI rules

React 18 + TypeScript (strict) + Vite + Tailwind. Talks to the Go backend in `../backend` over cookie sessions. Follow these when adding or changing UI code.

## Structure

- `pages/` are routes (one per sidebar item, plus `WalletDetail`). `components/` are pieces; group by area (`wallet/`, `send/`, `swap/`, `workspace/`, `transaction/`, `approvals/`) once an area has several. `components/ui/` holds design-system primitives (`Tooltip`, `StatusLabel`); generic pieces used across areas (`Modal`, `EmptyState`) stay at the top.
- `hooks/` hold stateful logic shared by several components or complex enough to test on its own (`useWallets`, `useSendForm`). Logic used by one component and simple stays in that component.
- `lib/` is pure helpers with no React. `types.ts` holds API types; `api.ts` is the only place that calls `fetch`.
- `AppShell` owns app-wide state (workspaces, wallets, live events) and passes it down. Don't add a global store or context for state one parent can own.

## API calls

- Every request goes through `api.ts` `req()`: it sends cookies, the `X-CSRF-Protection` header, and unwraps `{data}` / `{error}`. Never call `fetch` from components.
- Errors are `ApiError` with `status` and `code`. Branch on `code` (see `lib/authErrors.ts`), never on message text. A `401` ends the session through the `onUnauthorized` callback.
- Show `error.message` from the backend to the user; it is already written for them. Use a fallback only when the error isn't an `ApiError`.
- Workspace-scoped requests send the workspace id explicitly (`api.listWallets(workspaceId)`). The session's active workspace is shared by all tabs, so never rely on it for reads or writes.
- Add the TypeScript type for a new response to `types.ts` and keep field names identical to the backend JSON.

## State and loading

- Never render an empty state until the data has actually loaded. Track `loaded` (and `failed`) per request, keyed by what was requested (`loadedFor === workspaceId`), so a switch doesn't show the previous workspace's data.
- A failed request is not an empty list: show an error with **Try again**, not "Create your first …".
- Gate the whole shell on data it needs to choose a screen (workspaces before the picker or home) so the UI doesn't flash one screen then another.
- Fetch in a parent that stays mounted, not in components inside tabs, conditional panels or modals: those remount on every switch, refetch, and flash a loading state. Pass the loaded data down.
- When a component may decide to render nothing once data arrives, render nothing until it has loaded, not a loading placeholder that then disappears.
- Hooks are called before any early `return`.
- `localStorage` / `sessionStorage` can throw (private mode, blocked storage): wrap every access in `try/catch` and keep working without it.

## Permissions

- Hide actions the user can't perform instead of letting the backend reject them: members don't see Send or approval actions; only owners see create/delete wallet, Swap and add-asset. Pass optional callbacks (`onCreate?`, `onDelete?`) or a `readOnly` flag; don't re-derive roles inside leaf components.
- The backend still enforces every rule; hiding is for clarity, not security.

## Styling

- Tailwind utilities with the theme tokens in `tailwind.config.js`: `ink`, `ink-soft`, `muted`, `page`, `line`/`line-strong`, `card`, `hover`, `brand`/`brand-deep`/`brand-soft`, `danger`, `warning`, `success`. No raw hex colors (`bg-[#fdecec]` is `bg-danger-soft`, `text-[#d33a3a]` is `text-danger`).
- Use the component classes in `styles.css`; don't hand-roll their look with utilities:
  - Buttons: `btn` plus one variant (`btn-primary`, `btn-secondary`, `btn-danger`, `btn-danger-soft`, `btn-ghost`, `btn-ghost-danger`, `btn-link`) and optionally `btn-sm` / `btn-lg`. One primary button per view; destructive confirms use `btn-danger`; inline text actions (Remove, Revoke) use `btn-ghost-danger`.
  - Labels: `badge` plus `badge-neutral` / `badge-brand` / `badge-success` / `badge-warning` / `badge-danger`. Numbers on nav items and tabs use `count`.
  - Status: `dot` plus `bg-success` / `bg-warning` / `bg-danger` / `bg-brand` / `bg-line-strong`, followed by a word.
  - People and workspaces: `avatar` (or `avatar avatar-brand` for workspaces).
  - Notices: `notice` plus `notice-info` / `notice-warning` / `notice-danger`.
  - Inputs `field-input`, cards `surface`.
- Tooltips use `components/ui/Tooltip.tsx`, never the `title` attribute (it is invisible to keyboard and touch users). Icon-only buttons still need an `aria-label`.
- No rounded corners and no shadows on cards; the design is square and flat. `rounded-full` is only for dots, toggles and spinners.
- Match existing patterns before inventing one: settings sections are `surface` cards with a `text-[17px] font-bold` heading; lists are rows with `border-b border-line last:border-none`; dialogs use `Modal`; login-style screens copy the `Login` layout.
- Status reads as a dot plus a word (green "Connected", grey "Unavailable"), not a coloured badge on every row.
- Layouts work from 360px wide: stack with `flex-col sm:flex-row`, truncate long emails and names with `min-w-0 truncate`.

## Accessibility

- Every input has a label tied to it (`<label>` wrapping only the label text and input, or `htmlFor`/`id`). Hints and errors go outside the label and are linked with `aria-describedby`; otherwise they become part of the field's name.
- Icon-only and ambiguous buttons get `aria-label` (`"Accept invitation to Treasury"`). Decorative SVGs get `aria-hidden`.
- Alerts use `role="alert"`, loading and status text use `role="status"`, tabs use `role="tablist"`/`tab`/`tabpanel` with `aria-selected`.
- Close menus and modals on Escape and outside click.
- Tests and screen readers rely on these names; prefer `getByRole`/`getByLabel` selectors over CSS ones in e2e.

## Routing

- Hash routing (`hooks/useRoute.ts`): `#/wallets`, `#/send`, `#/swap`, `#/approvals`, `#/approvals/:id`, `#/settings`, `#/wallet/:id`. Links that must survive sign-in (invites, approval requests) are captured on load and on `hashchange`, then stored in `sessionStorage` because the OAuth redirect drops the hash.

## Copy

- Write for the user, not the system: "Only workspace owners can send for now", not "403 forbidden".
- Buttons say what happens ("Accept invite", "Create workspace"); progress states use an ellipsis ("Joining…").

## Tests

- `npm test` runs the Node unit tests in `tests/`; `npm run build` must pass (it typechecks).
- `e2e/` is Playwright against a live backend and vite. Put shared setup in `e2e/helpers.ts`, create unique accounts per test, and set up data through the API rather than clicking through it. Never commit screenshots or specs with absolute paths.
- Run e2e with a backend config whose `auth_rate_limit` is high; the default (30/min) is below what the suites use.
- Fund test wallets with `fundAccount` (`e2e/fund.ts`), never Friendbot: it sends XLM from a plain testnet account whose secret is in the git-ignored `e2e/.fund-wallet.json`. Set it up once with `npm run e2e:fund -- create`, and return leftover test XLM with `npm run e2e:fund -- sweep <since>`.
- Specs that create wallets run with `test.describe.configure({ mode: 'serial' })` and wait up to 3 minutes for keygen: the local cluster queues keygens at about 10 s each.

## Comments and commits

- Follow the workspace `CLAUDE.md`: no narration comments, one short line at most, only for a non-obvious why.
- Never add `Co-Authored-By` or any trailer, and no tool footer, to commits.
