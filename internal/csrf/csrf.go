package csrf

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
)

const (
	CookieName    = "csrf_token"
	FormFieldName = "csrf_token"
	tokenLength   = 32
)

func GenerateToken() (string, error) {
	b := make([]byte, tokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func SetCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// ValidateToken validates the double-submit CSRF token: the value in the
// request cookie must be non-empty and match the form field value.
// Assumes r.ParseForm() has already been called.
func ValidateToken(r *http.Request) bool {
	cookie, err := r.Cookie(CookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	formToken := r.FormValue(FormFieldName)
	if formToken == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(formToken), []byte(cookie.Value)) == 1
}
