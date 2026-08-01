CREATE TYPE referral_urgency AS ENUM ('routine', 'urgent', 'emergency');
CREATE TYPE referral_status AS ENUM ('created', 'routed', 'accepted', 'declined', 'in_progress', 'completed', 'cancelled');

CREATE TABLE referrals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id UUID NOT NULL REFERENCES patients (id),
    referring_provider_id UUID NOT NULL REFERENCES providers (id),
    referring_facility_id UUID NOT NULL REFERENCES facilities (id),
    receiving_facility_id UUID REFERENCES facilities (id),
    receiving_provider_id UUID REFERENCES providers (id),
    specialty_requested TEXT NOT NULL,
    urgency referral_urgency NOT NULL DEFAULT 'routine',
    status referral_status NOT NULL DEFAULT 'created',
    reason TEXT NOT NULL,
    clinical_summary_encounter_id UUID REFERENCES encounters (id),
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_referrals_patient ON referrals (patient_id);
CREATE INDEX idx_referrals_receiving_facility ON referrals (receiving_facility_id);
CREATE INDEX idx_referrals_status ON referrals (status);

CREATE TABLE referral_status_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    referral_id UUID NOT NULL REFERENCES referrals (id),
    from_status referral_status,
    to_status referral_status NOT NULL,
    actor_user_id UUID NOT NULL REFERENCES users (id),
    note TEXT,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_referral_status_events_referral ON referral_status_events (referral_id);
