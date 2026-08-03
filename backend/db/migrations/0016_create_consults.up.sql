-- consultations is the direct provider-to-provider "second opinion" flow:
-- one named provider asks another named provider about one patient.
-- Unlike referrals (which transfer a patient to a whole receiving
-- FACILITY and need a consent-grant-table entry plus a 7-state workflow),
-- a consult never grants access via the consent module — authorization is
-- just "the two named providers, or system_admin" — and has only 3
-- states. See internal/consults for the party-check logic.
CREATE TYPE consult_status AS ENUM ('requested', 'responded', 'closed');

CREATE TABLE consultations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id UUID NOT NULL REFERENCES patients (id),
    requesting_provider_id UUID NOT NULL REFERENCES providers (id),
    target_provider_id UUID NOT NULL REFERENCES providers (id),
    reason TEXT NOT NULL,
    status consult_status NOT NULL DEFAULT 'requested',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_consultations_patient ON consultations (patient_id);
CREATE INDEX idx_consultations_requesting_provider ON consultations (requesting_provider_id);
CREATE INDEX idx_consultations_target_provider ON consultations (target_provider_id);

-- attachment_ids deliberately has no FK constraint (an array column can't
-- FK-reference cleanly) — application layer validates that any id passed
-- in belongs to the same patient_id as the consultation, see
-- consults.Service.PostMessage.
CREATE TABLE consultation_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    consultation_id UUID NOT NULL REFERENCES consultations (id),
    sender_id UUID NOT NULL REFERENCES users (id),
    body TEXT NOT NULL,
    attachment_ids UUID[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_consultation_messages_consultation ON consultation_messages (consultation_id);
