# Frontend organization

- `App.vue` composes domain views and dialog contents, and connects their typed events to workspace actions.
- `app/` owns cross-domain coordination: navigation, the API snapshot, polling, and mutation feedback. `useWorkspace` creates state per application instance.
- `domains/` groups features by business responsibility:
  - `auth`: Google sign-in UI, cookie-session restoration, CSRF headers, user profiles, and optional local-admin login.
  - `devices`: enrollment, credentials, device lists/details, telemetry, filtering, hardware definitions, and device API calls.
  - `firmware`: artifact cards, uploads, and firmware API calls.
  - `deployments`: compatible-device selection, job history, job status rules, and deployment API calls.
  - `overview`: fleet statistics and recent activity.
  - `connection`: the hardware connection guide.
- `components/layout/` contains the application shell and sidebar.
- `components/ui/` contains reusable, domain-independent UI such as the modal, status badge, and toast.
- `shared/` contains the authenticated HTTP transport, formatting helpers, and notification lifecycle.
- `styles/main.css` contains the shared blue theme, component classes, and responsive rules.

## Component boundaries

Domain components receive data through typed props and emit user intents. Forms own their draft state and reset when their dialog is reopened. Device filters are kept alive while navigating between workspace pages. Server data stays in the workspace snapshot so polling updates all visible views consistently.

Keep feature-specific types, API calls, helpers, and components in their domain. Cross-domain workflows belong in `app/`; shared UI and transport should not import domain state. Domain API factories receive the HTTP client instead of reading session storage themselves.

Run `bun run build` for TypeScript and production-build checks, and `bun run test:e2e` for browser workflows.
