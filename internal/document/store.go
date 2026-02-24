package document

import "sync"

// Document holds the text content of an open file.
type Document struct {
	URI     string
	Content string
}

// Store is a thread-safe map from document URI to Document.
type Store struct {
	mu   sync.RWMutex
	docs map[string]*Document
}

// New returns an initialized Store.
func New() *Store {
	return &Store{docs: make(map[string]*Document)}
}

// Open stores a newly opened document.
func (s *Store) Open(uri, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs[uri] = &Document{URI: uri, Content: text}
}

// Update replaces the content of an existing document.
func (s *Store) Update(uri, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if doc, ok := s.docs[uri]; ok {
		doc.Content = text
	} else {
		s.docs[uri] = &Document{URI: uri, Content: text}
	}
}

// Close removes a document from the store.
func (s *Store) Close(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.docs, uri)
}

// Get retrieves a document by URI. Returns ("", false) if not found.
func (s *Store) Get(uri string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	doc, ok := s.docs[uri]
	if !ok {
		return "", false
	}
	return doc.Content, true
}
