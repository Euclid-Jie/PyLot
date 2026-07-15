package commandline

import "testing"

func TestParseCustomPythonModuleCommand(t *testing.T) {
	exe, args, err := Parse(`.venv\Scripts\python.exe -m fof99_nav_ingestion run --include-stale --update-token`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if exe != `.venv\Scripts\python.exe` {
		t.Fatalf("exe = %q", exe)
	}
	want := []string{"-m", "fof99_nav_ingestion", "run", "--include-stale", "--update-token"}
	if len(args) != len(want) {
		t.Fatalf("args length = %d, want %d: %#v", len(args), len(want), args)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("args[%d] = %q, want %q", i, args[i], want[i])
		}
	}
}

func TestParseArgsKeepsQuotedValue(t *testing.T) {
	args, err := ParseArgs(`--name "alpha beta" --flag`)
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}
	want := []string{"--name", "alpha beta", "--flag"}
	if len(args) != len(want) {
		t.Fatalf("args length = %d, want %d: %#v", len(args), len(want), args)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("args[%d] = %q, want %q", i, args[i], want[i])
		}
	}
}
