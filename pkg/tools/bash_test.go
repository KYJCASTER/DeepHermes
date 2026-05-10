package tools

import "testing"

func TestNormalizeWindowsPowerShellCommandReplacesAndAndOutsideQuotes(t *testing.T) {
	input := `pwd && dir && Write-Output "a && b" && Write-Output 'c && d'`
	want := `pwd ; dir ; Write-Output "a && b" ; Write-Output 'c && d'`

	if got := normalizeWindowsPowerShellCommand(input); got != want {
		t.Fatalf("unexpected normalized command:\nwant %q\n got %q", want, got)
	}
}

func TestNormalizeWindowsPowerShellCommandPreservesCommandsWithoutAndAnd(t *testing.T) {
	input := `Get-ChildItem -Force`
	if got := normalizeWindowsPowerShellCommand(input); got != input {
		t.Fatalf("expected command to be unchanged, got %q", got)
	}
}
