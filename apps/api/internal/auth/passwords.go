package auth

// Passwords is the Argon2id implementation of services.Passwords.
type Passwords struct{}

func (Passwords) Hash(password string) string {
	return HashPassword(password)
}

func (Passwords) Verify(password, encoded string) (bool, error) {
	return VerifyPassword(password, encoded)
}

func (Passwords) VerifyDummy(password string) {
	VerifyDummy(password)
}
