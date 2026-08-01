package platform

import "errors"

// IsUniqueViolation reports whether err is a Postgres unique-constraint
// violation (SQLSTATE 23505) — every module's Create* repository method
// needs this to translate a raw DB error into a domain "already exists"
// error.
func IsUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}
