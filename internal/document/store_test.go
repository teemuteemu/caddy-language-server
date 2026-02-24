package document

import (
	"sync"
	"testing"
)

func TestStore_OpenAndGet(t *testing.T) {
	s := New()
	s.Open("file:///test.caddyfile", "example.com {}")
	got, ok := s.Get("file:///test.caddyfile")
	if !ok {
		t.Fatal("Get returned ok=false after Open")
	}
	if got != "example.com {}" {
		t.Errorf("got %q, want %q", got, "example.com {}")
	}
}

func TestStore_GetMissing(t *testing.T) {
	s := New()
	_, ok := s.Get("file:///nonexistent.caddyfile")
	if ok {
		t.Error("Get returned ok=true for non-existent document")
	}
}

func TestStore_Update(t *testing.T) {
	s := New()
	s.Open("file:///test.caddyfile", "original")
	s.Update("file:///test.caddyfile", "updated")
	got, ok := s.Get("file:///test.caddyfile")
	if !ok {
		t.Fatal("Get returned ok=false after Update")
	}
	if got != "updated" {
		t.Errorf("got %q, want 'updated'", got)
	}
}

func TestStore_UpdateCreatesIfMissing(t *testing.T) {
	// Update must behave like Open when the document does not exist yet.
	s := New()
	s.Update("file:///new.caddyfile", "content")
	got, ok := s.Get("file:///new.caddyfile")
	if !ok {
		t.Fatal("Get returned ok=false after Update on new document")
	}
	if got != "content" {
		t.Errorf("got %q, want 'content'", got)
	}
}

func TestStore_Close(t *testing.T) {
	s := New()
	s.Open("file:///test.caddyfile", "content")
	s.Close("file:///test.caddyfile")
	_, ok := s.Get("file:///test.caddyfile")
	if ok {
		t.Error("Get returned ok=true after Close")
	}
}

func TestStore_CloseNonExistent(t *testing.T) {
	// Closing a document that was never opened must not panic.
	s := New()
	s.Close("file:///ghost.caddyfile")
}

func TestStore_OpenOverwrites(t *testing.T) {
	// Opening the same URI twice should replace the content.
	s := New()
	s.Open("file:///test.caddyfile", "first")
	s.Open("file:///test.caddyfile", "second")
	got, _ := s.Get("file:///test.caddyfile")
	if got != "second" {
		t.Errorf("got %q, want 'second'", got)
	}
}

func TestStore_MultipleDocuments(t *testing.T) {
	s := New()
	s.Open("file:///a.caddyfile", "aaa")
	s.Open("file:///b.caddyfile", "bbb")

	a, ok := s.Get("file:///a.caddyfile")
	if !ok || a != "aaa" {
		t.Errorf("document a: got (%q, %v), want ('aaa', true)", a, ok)
	}
	b, ok := s.Get("file:///b.caddyfile")
	if !ok || b != "bbb" {
		t.Errorf("document b: got (%q, %v), want ('bbb', true)", b, ok)
	}
}

func TestStore_ConcurrentReadWrite(t *testing.T) {
	// Exercise the RWMutex under concurrent load. Any data race will be caught
	// by the race detector (go test -race).
	s := New()
	s.Open("file:///test.caddyfile", "initial")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(3)
		go func(i int) {
			defer wg.Done()
			s.Update("file:///test.caddyfile", "updated")
		}(i)
		go func() {
			defer wg.Done()
			s.Get("file:///test.caddyfile")
		}()
		go func() {
			defer wg.Done()
			s.Get("file:///other.caddyfile")
		}()
	}
	wg.Wait()
}
