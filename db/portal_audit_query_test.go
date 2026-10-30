package db

import "testing"

func TestLikeEscapeKeepsWildcardsLiteral(t *testing.T) {
	if got := likeEscape(`50%_off\`); got != `50\%\_off\\` {
		t.Errorf("likeEscape = %q", got)
	}
}
