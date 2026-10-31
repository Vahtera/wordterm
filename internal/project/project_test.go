package project

import "testing"

func TestValidateName(t *testing.T) {
	bad := []string{"", " a", "a ", ".", "..", "a/b", `a\b`, "a:b", "con", "NUL.txt", "x.", "a*b"}
	for _, n := range bad {
		if ValidateName(n) == nil {
			t.Errorf("%q should be invalid", n)
		}
	}
	for _, n := range []string{"Chapter 1", "Kappale ääkköset", "章 1", "a.b"} {
		if err := ValidateName(n); err != nil {
			t.Errorf("%q should be valid: %v", n, err)
		}
	}
}

func TestCreateLoadResolve(t *testing.T) {
	parent := t.TempDir()
	dir, err := Create(parent, "Novel", "Me", "")
	if err != nil {
		t.Fatal(err)
	}
	p, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ref, ok := p.Meta.Resolve(); !ok || ref.Scene != "Scene1" {
		t.Fatalf("resolve = %v %v", ref, ok)
	}
	p.Meta.LastOpened = &SceneRef{"Gone", "Gone"}
	if ref, ok := p.Meta.Resolve(); !ok || ref.Chapter != "Chapter1" {
		t.Fatal("fallback to first scene failed")
	}
	if _, err := Create(parent, "Novel", "Me", ""); err == nil {
		t.Fatal("Create must refuse to overwrite an existing project")
	}
}

func TestCountWords(t *testing.T) {
	if n := CountWords("héllo wörld  日本\t語\n x"); n != 5 {
		t.Fatalf("got %d, want 5", n)
	}
}
