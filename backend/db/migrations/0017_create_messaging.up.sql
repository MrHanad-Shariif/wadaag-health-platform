-- Secure messaging (Phase 5): general-purpose two-way communication between
-- any two (or more) users, not scoped to a single patient the way
-- records/referrals/consults are. Authorization is simply "is the caller a
-- member of this conversation" — see internal/messaging for why this
-- deliberately does not go through consent.Middleware or audit.Logger.
CREATE TYPE conversation_type AS ENUM ('direct', 'group');

CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type conversation_type NOT NULL,
    created_by UUID NOT NULL REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE conversation_members (
    conversation_id UUID NOT NULL REFERENCES conversations (id),
    user_id UUID NOT NULL REFERENCES users (id),
    last_read_at TIMESTAMPTZ,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (conversation_id, user_id)
);

CREATE INDEX idx_conversation_members_user ON conversation_members (user_id);

-- attachment_ids deliberately has no FK constraint, same as
-- consultation_messages — see consults' migration comment. Unlike consults,
-- there is no single patient_id to validate attachment ownership against
-- here, so this pass accepts attachment_ids as-is with no validation at all
-- (see internal/messaging/service.go doc comment).
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations (id),
    sender_id UUID NOT NULL REFERENCES users (id),
    body TEXT NOT NULL,
    attachment_ids UUID[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_messages_conversation ON messages (conversation_id, created_at);
