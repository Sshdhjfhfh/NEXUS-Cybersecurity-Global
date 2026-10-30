// Portal password rules, following NIST SP 800-63B: a length range, no
// composition rules, but refuse passwords that are common, trivially
// predictable, or built from the account's own email.
package server

import (
	"strings"
	"unicode"
)

const (
	errPasswordLength = "la contraseña debe tener entre 8 y 72 caracteres"
	errPasswordCommon = "esa contraseña es demasiado común o predecible; elige otra"
	errPasswordEmail  = "la contraseña no puede contener tu email"
)

// commonPasswords are among the most used (and first guessed) passwords in
// public breach corpora, plus local variants. Matched case-insensitively, both
// whole and with trailing digits/symbols removed ("Password2026!" -> "password").
var commonPasswords = func() map[string]bool {
	m := map[string]bool{}
	for _, p := range strings.Fields(`
		123456 1234567 12345678 123456789 1234567890 12345678910 0123456789 987654321 87654321
		111111 11111111 000000 00000000 123123 123123123 112233 121212 123321 654321 666666 777777 7777777
		password passw0rd p@ssw0rd p@ssword pa55word passwort motdepasse contrasena contraseña senha
		qwerty qwertyui qwertyuiop qwertz azerty asdfgh asdfghjk asdfghjkl zxcvbn zxcvbnm qazwsx 1qaz2wsx
		1q2w3e4r 1q2w3e4r5t 1q2w3e4r5t6y q1w2e3r4 qwe123 qweasd qweasdzxc abc123 abcd1234 abc12345 a1b2c3d4
		iloveyou letmein welcome admin administrator root toor changeme secret default login guest test
		master dragon monkey shadow sunshine princess football baseball soccer hockey superman batman
		starwars trustno1 freedom whatever michael jennifer jessica charlie computer internet samsung
		qwerty123 password1 password123 welcome1 admin123 letmein1 iloveyou1 test123 test1234
		teamo tequiero holamundo futbol barcelona realmadrid espana españa mexico argentina colombia
		nexus nexusia cybersecurity ciberseguridad seguridad security
	`) {
		m[p] = true
	}
	return m
}()

// portalPasswordProblem returns why a new password is refused ("" when it is
// acceptable). email is the account's address (may be empty when unknown).
func portalPasswordProblem(pw, email string) string {
	if len(pw) < 8 || len(pw) > 72 {
		return errPasswordLength
	}
	low := strings.ToLower(pw)
	base := strings.TrimRightFunc(low, func(r rune) bool { return !unicode.IsLetter(r) })
	if commonPasswords[low] || (len(base) >= 4 && commonPasswords[base]) || predictable(low) {
		return errPasswordCommon
	}
	if email = strings.ToLower(strings.TrimSpace(email)); email != "" {
		local, _, _ := strings.Cut(email, "@")
		if strings.Contains(low, email) || (len(local) >= 4 && strings.Contains(low, local)) {
			return errPasswordEmail
		}
	}
	return ""
}

// predictable reports a password made of one repeated character, or a run that
// only steps up or down by one ("abcdefgh", "98765432"), or a keyboard row.
func predictable(s string) bool {
	r := []rune(s)
	same, up, down := true, true, true
	for i := 1; i < len(r); i++ {
		same = same && r[i] == r[0]
		up = up && r[i] == r[i-1]+1
		down = down && r[i] == r[i-1]-1
	}
	if same || up || down {
		return true
	}
	for _, row := range []string{"1234567890", "qwertyuiop", "asdfghjkl", "zxcvbnm"} {
		if strings.Contains(row, s) || strings.Contains(reverse(row), s) {
			return true
		}
	}
	return false
}

func reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}
