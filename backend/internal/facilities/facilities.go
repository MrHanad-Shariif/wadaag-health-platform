package facilities

import (
	"time"

	"github.com/google/uuid"
)

type Type string

const (
	TypeHospital Type = "hospital"
	TypeClinic   Type = "clinic"
	TypeLab      Type = "lab"
	TypePharmacy Type = "pharmacy"
	TypeInsurer  Type = "insurer"
)

type Facility struct {
	ID        uuid.UUID
	Name      string
	Type      Type
	Region    *string
	District  *string
	Phone     *string
	Address   *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Provider struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	FacilityID    uuid.UUID
	Specialty     *string
	LicenseNumber *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
