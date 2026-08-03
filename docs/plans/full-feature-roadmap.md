# Full Feature Roadmap — System Design

Owner-provided scope, reviewed against the current codebase on 2026-08-02. This
supersedes nothing already built; it fills the gaps and sequences what's left.

## Method note

Referenced conventions while designing this: FHIR's `ServiceRequest` resource
(the merged successor to `ReferralRequest`/`ProcedureRequest` — HL7 dropped the
distinction because a referral and a "please review this case" consultation
request are the same shape with a different `intent`/`code`), FHIR `Encounter`
for visit modeling, and OpenMRS's privilege/role split (privileges are atomic
`resource:action`-style grants, roles are named bundles of them — this is
already this repo's dynamic `authz` model, not the legacy fixed-enum one).
Deliberately **not** importing FHIR's full resource complexity (references,
extensions, terminology bindings) — this system needs FHIR-shaped data, not a
FHIR server, at its current stage.

## Current state (verified against the code, not assumed)

**Built:** `identity` (self-register as `patient` w/ own bcrypt password,
login issuing access+refresh JWTs, `/me` — **no `/refresh` or `/logout`
endpoint despite refresh tokens being issued**), `authz` (dynamic
roles/permissions, admin-driven user creation), `facilities` (hospital CRUD,
backend only — **no frontend admin pages**, confirmed only `types.ts`/`api.ts`
exist under `web/src/features/facilities/`), `records` (patients, encounters,
clinical observations with a `vitals|diagnosis|note|attachment` type enum —
**no real file storage**, `attachment` observations have nowhere to point),
`referrals` (full create→accept/decline/start/complete/cancel lifecycle with
status-event timeline, consent-gated), `consent` (per-patient access grants),
`audit` (append-only log, wired into records/referrals only), `dashboard`
(patient/encounter/facility/user counts + referrals-by-status).

**Confirmed empty** (not stubbed with code, just absent):
`backend/internal/{consults,insurance,lab,notifications,pharmacy,sync,telemedicine}`
— the README's "reserved" language overstates what's there; these are
greenfield.

**Missing entirely:** doctor profile fields (license #, specialization,
qualifications, years of experience, verification status, certificates, areas
of expertise — providers today only have free-text `specialty`/
`license_number` from `facilities.CreateProviderInput`), patient medical
history (allergies, chronic conditions, medications, surgeries, family
history, vaccinations), hospital branches/departments/working hours/logo,
doctor consultation flow, messaging, notification delivery (in-app, email,
push), real attachment storage, search, appointments, reports beyond the
basic dashboard, a unified admin dashboard, settings pages, forgot/reset
password, email verification, session/device management, rate limiting,
login-attempt lockout.

## Backlog — not building now (owner marked these future/out of scope)

- Two-factor authentication
- Nurse and Receptionist roles
- SMS notifications
- Telemedicine (all of section 14 — video, voice, screen share, session notes)

## Needs a decision from the product owner before implementation

These need a real account/API key/infra choice — I'm not picking a paid
vendor silently:

1. **File storage backend for medical attachments** — local disk (simplest,
   fine for single-host MVP, no offsite durability) vs. S3-compatible object
   storage (DigitalOcean Spaces / AWS S3 / MinIO self-hosted — durable,
   needed once this runs on more than one host, needs credentials).
2. **Email delivery provider** — for verification, password reset, and email
   notifications (e.g. Postmark, SES, SendGrid — needs an account + API key +
   a verified sending domain).
3. **Push notification provider** — for browser/mobile push (e.g. Firebase
   Cloud Messaging — needs a project + credentials; mobile app is a later
   phase per the repo's own README, so this can wait until then without
   blocking anything web-facing).

Everything else below is a technical call I'm making as part of the design,
not a product decision: real-time transport for messaging/notifications is
**WebSocket with a polling fallback** (`nhooyr.io/websocket` or `gorilla/
websocket`, one hub per facility/conversation), not a third-party realtime
service — keeps the stack dependency-free and self-hostable, consistent with
the rest of this repo.

## Phases

Each phase lists backend (module, migration, endpoints) and frontend (pages/
components) scope. Phases are ordered by dependency — later phases assume
earlier ones landed. Within a phase, backend and frontend tasks for the same
capability are parallelizable once the API contract is fixed; cross-phase
work is not (see task graph for exact `blocks`/`blockedBy`).

### Phase 0 — Auth & Session Hardening
Blocks almost everything else being safe to run for real users.
- Migration `0007_create_sessions.up/down.sql`: `refresh_tokens` table
  (id, user_id, token_hash, device_label, ip, user_agent, created_at,
  expires_at, revoked_at) so refresh/logout/device-management have something
  to act on — JWT refresh tokens today are stateless and unrevokable.
- `identity`: `POST /auth/refresh` (rotate refresh token, verify against
  `refresh_tokens`, issue new access token), `POST /auth/logout` (revoke the
  presented refresh token), `GET /auth/sessions` + `DELETE /auth/sessions/{id}`
  (device management — list/revoke other active sessions).
- `identity`: `POST /auth/forgot-password` (issue a short-lived reset token,
  email it — depends on email provider decision above; until that's decided,
  land the endpoint logging the token instead of emailing it, so the rest of
  the flow is testable), `POST /auth/reset-password`.
- `identity`: email verification — `verified_at` column on `users`
  (migration `0008`), `POST /auth/send-verification`,
  `GET /auth/verify-email?token=...`.
- `platform`: login-attempt rate limiting + lockout (in-memory or DB-backed
  counter keyed by identifier+IP, lock after N failures for a cooldown), and
  a general per-IP rate limit middleware applied at the router level.
- Frontend: forgot/reset password pages, email-verification landing page, a
  "Sessions" panel under Settings (Phase 9) — stub the page now, wire it once
  Settings lands.

### Phase 1 — User Profiles & RBAC Expansion
- Migration `0009_create_user_profiles.up/down.sql`: `user_profiles` table
  (user_id FK, photo_url, bio, languages text[], availability_status,
  is_online, last_seen_at) — separate table, not columns on `users`, since
  `identity` and profile-editing are different concerns with different write
  patterns.
- `identity` (or a new `profiles` sub-module if `identity` gets crowded):
  `GET/PATCH /me/profile`, `POST /me/profile/photo`.
- RBAC: add `Super Admin`, `Organization Admin`, `Specialist` as **dynamic**
  `authz` roles (not new legacy enum values) — the legacy fixed-role enum
  (`platform.Role`) is load-bearing for `RequireRoles` across records/
  referrals/facilities and adding roles there means touching every gate;
  the dynamic system exists exactly for "new role, new permission bundle"
  without a code change. Map: Super Admin = `FullAccess` (same as today's
  seeded Administrator), Organization Admin = facility-scoped full access
  (new permission scope concept — see note below), Specialist = same
  permission bundle as Doctor plus `consultations:accept`.
  **Design note:** today's dynamic permissions are global (`resource:action`,
  no facility scoping) — Organization/Hospital Admin needs facility-scoped
  permissions eventually. For this phase, ship Organization Admin as
  functionally identical to Hospital Admin (facility-affiliated via the
  existing `facilities.CreateProviderInput` link) and revisit true
  multi-branch scoping in Phase 2 once branches exist.
- Extend the permission catalog: `hospitals:manage`, `reports:view` (partially
  covered by existing `users`/`roles` permissions — add the missing pairs
  from the owner's list: view/edit patients, create/accept referrals, upload
  records, manage hospitals, view reports).
- Frontend: profile edit page, online/offline indicator in `UserMenu`, role
  dropdown in `CreateUserPage`/role admin pages picks up new dynamic roles
  automatically (no frontend change needed there — verify).

### Phase 2 — Hospital, Doctor & Patient Data Completeness
- Migration `0010_extend_facilities.up/down.sql`: `branches` table (parent
  facility_id, name, address, phone), `departments` table (facility_id,
  name), `logo_url`/`working_hours` (jsonb) columns on `facilities`.
- Migration `0011_extend_providers.up/down.sql`: on the existing providers
  table, add `qualifications text[]`, `years_experience`, `consultation_fee`,
  `verification_status`, `languages text[]`, `certificates jsonb`,
  `areas_of_expertise text[]` (specialty/license_number already exist).
- Migration `0012_extend_patients.up/down.sql`: add `gender`, `blood_group`
  to `patients`; new `patient_medical_history` table (allergies, chronic
  conditions, current medications, past surgeries, family history,
  vaccination history — jsonb arrays, one row per patient, editable by
  physician/hospital_admin with the same consent gate as encounters).
- `facilities`: branch/department CRUD endpoints, `PATCH /facilities/{id}`
  for logo/working-hours.
- `records` or a split `providers` module: provider profile CRUD, medical
  history CRUD.
- Frontend: **hospital admin pages don't exist yet at all** — facility list/
  detail/create, branch/department management, doctor profile page (view +
  self-edit), patient medical-history section on `PatientDetailPage`.

### Phase 3 — EMR Expansion & Real Medical Attachments
- Migration `0013_extend_observations.up/down.sql` or dedicated tables for
  `prescriptions`, `lab_results`, `radiology_reports` if their structure
  diverges enough from generic observations to need typed columns (dosage/
  frequency for prescriptions, reference ranges for lab results) — use
  typed tables, not just more `ObservationType` enum values, once a type
  needs its own queryable fields.
- Migration `0014_create_attachments.up/down.sql`: `attachments` table
  (patient_id, encounter_id nullable, uploaded_by, file_key, filename,
  mime_type, size, version, created_at) — `file_key` points at whichever
  storage backend gets decided (local path or object-store key, same column
  either way).
- New `records` sub-feature or new `attachments` module: `POST /attachments`
  (multipart upload — blocked on storage-backend decision), `GET
  /attachments/{id}` (streams/redirects for preview or download),
  `GET /attachments/{id}/versions`, permission check reusing the existing
  consent middleware pattern.
- Frontend: attachment upload widget on encounter/patient pages, preview
  (PDF/image inline, others as download), version history list.

### Phase 4 — Doctor Consultation ("second opinion") Flow
- New `consults` module (currently empty). Migration
  `0015_create_consults.up/down.sql`: `consultations` (patient_id,
  requesting_provider_id, target_provider_id, reason, status, created_at),
  `consultation_messages` (consultation_id, sender_id, body, attachment_ids,
  created_at) — this is the FHIR-`ServiceRequest`-with-`intent=consult`
  shape, deliberately structured like `referrals` since it's the same
  "one clinician asks another" pattern with lighter-weight resolution
  (no facility transfer, no accept/decline/start/complete state machine —
  just requested → responded → closed).
- `consults`: `POST /consults`, `GET /consults/inbox`,
  `POST /consults/{id}/messages`, `GET /consults/{id}`,
  `POST /consults/{id}/close`.
- Frontend: consultation inbox (mirrors `ReferralsInboxPage`), new
  consultation form, consultation thread view.

### Phase 5 — Secure Messaging
- New `messaging` module. Migration `0016_create_messaging.up/down.sql`:
  `conversations` (type: direct|group, created_by), `conversation_members`
  (conversation_id, user_id, last_read_at), `messages` (conversation_id,
  sender_id, body, attachment_ids, created_at).
- `messaging`: REST for history/send (`GET/POST /conversations`,
  `GET/POST /conversations/{id}/messages`) plus a WebSocket endpoint
  (`/ws/conversations/{id}`) for live delivery, typing indicators, read
  receipts — see the realtime-transport call above.
- Voice notes = attachments with `mime_type` audio/*, reusing Phase 3's
  attachment pipeline rather than a separate system.
- Frontend: conversation list, thread view with typing indicator/read
  receipts, message search (defer full-text search itself to Phase 7 and
  reuse it here).

### Phase 6 — Notifications
- New `notifications` module. Migration `0017_create_notifications.up/
  down.sql`: `notifications` table (user_id, type, payload jsonb, read_at,
  created_at) for the in-app feed; a `notification_preferences` table
  (user_id, channel, enabled) for per-channel opt-in/out (Settings, Phase 9).
- Event sources: hook into `referrals`, `consults` (Phase 4), `messaging`
  (Phase 5), `records` (patient updated) to write a notification row on the
  relevant transitions — a small internal `Publish(ctx, userID, type,
  payload)` interface each module calls, not a shared event bus (no message
  queue in this stack yet, and one isn't justified by this volume).
- Email/push delivery are adapters behind that same interface, gated on the
  provider decisions above — land the in-app feed first since it has no
  external dependency, add email/push adapters once those are chosen.
- Frontend: notification bell + dropdown in `Layout.tsx`'s topbar, mark-read,
  notification preferences section in Settings.

### Phase 7 — Search
- Start with Postgres full-text search (`tsvector` columns +
  `pg_trgm`/GIN indexes on patients.full_name, providers, facilities,
  referrals, consultations, records) — no external search service needed at
  this scale.
- `search` endpoint(s) per the owner's list of facets (doctor, patient,
  hospital, specialty, disease, referral ID, consultation, medical record) —
  a single `GET /search?q=&type=&filters=` fanning out to per-domain
  repositories server-side, rather than one giant cross-table query.
- Saved searches: small `saved_searches` table (user_id, query, filters
  jsonb, name).
- Frontend: global search bar (topbar), results page with type filters,
  saved-search management in Settings.

### Phase 8 — Appointments (owner marked optional for MVP — lowest priority of the "now" work)
- New `appointments` module. Migration `0018_create_appointments.up/
  down.sql`: `appointments` (patient_id, provider_id, facility_id, start_at,
  end_at, status, reminder_sent_at), `provider_availability` (provider_id,
  weekday, start_time, end_time) feeding a simple slot-generation function
  rather than a full calendaring engine.
- `appointments`: CRUD + reschedule/cancel, availability query endpoint.
- Reminders reuse the Phase 6 notification pipeline (email/in-app), not a
  separate scheduler — a lightweight cron-style ticker checking upcoming
  `start_at` is enough at this volume.
- Frontend: calendar view, booking flow, availability editor for providers.

### Phase 9 — Reports & Analytics, Admin Dashboard, Settings
- `dashboard`: extend `Summary` with referral success rate, most-active-
  doctors, department stats (hospital view) and my-referrals/pending-
  consultations/patients-today (doctor view) — mostly aggregation queries
  against tables that exist by this point.
- New unified **Admin Dashboard** frontend area consolidating what's
  currently scattered (`authz`'s user/role pages, `facilities` once Phase 2
  ships, audit log viewer — `audit` has no read endpoint yet, add
  `GET /audit?filters=`) plus system settings and feature flags (a simple
  `feature_flags` table + admin toggle UI — no need for a full flag service
  at this scale).
- **Settings** pages (new frontend area, mostly wiring already-built
  endpoints): user settings (language, theme — theme already exists via
  `useTheme`, wire language + notification prefs + password change +
  profile edit into one place instead of scattered pages); organization
  settings (logo/branding/departments/working-hours from Phase 2, plus a new
  `referral_policies` concept — start as free-text policy notes on
  `facilities`, not a rules engine).

### Phase 10 — Security Hardening Pass
Cross-cutting, done last so it hardens the full surface area rather than
each phase re-deriving it: encryption at rest (Postgres-level, e.g. managed
DB encryption or `pgcrypto` for the most sensitive columns — this is mostly
an infra/deploy config decision, not application code), confirm HTTPS
termination is documented for prod deploy (`deploy/` currently has no
TLS-terminating reverse proxy config — add one), file scanning on upload
(ClamAV or a hosted scanning API — flag as another vendor decision if a
hosted API is preferred over self-hosted ClamAV), session expiration
enforcement (already partially there via JWT TTLs — audit that refresh
rotation from Phase 0 doesn't silently extend sessions forever), and a
review pass confirming every new endpoint from Phases 0-9 is gated by the
correct RBAC check (`RequireRoles` or `RequirePermission`) — this is the
`system-architect` review/green-light step, not new build work.

## Task graph

Created via `TaskCreate`/`TaskUpdate` with `blocks`/`blockedBy` wiring
mirroring phase order and the parallelizable-within-a-phase note above —
see the live task list for current status; this document is the design of
record, the task list is the execution tracker.
