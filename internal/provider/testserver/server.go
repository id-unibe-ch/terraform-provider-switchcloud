// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package testserver provides an in-process HTTP server that simulates the
// SwitchCloud API for use in acceptance tests. Each test can spin up its own
// isolated Server instance, seed initial state, and inject per-route error
// behaviour without depending on any external process.
package testserver

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"
)

// Project mirrors the SwitchCloud API project structure.
type Project struct {
	Id                       string  `json:"id"`
	Name                     string  `json:"name"`
	Description              *string `json:"description,omitempty"`
	CustomerBillingReference *string `json:"customer_billing_reference,omitempty"`
	TenantId                 string  `json:"tenant_id"`
	Archived                 bool    `json:"archived"`
	ArchivedAt               string  `json:"archived_at,omitempty"`
	CreatedAt                string  `json:"created_at"`
	UpdatedAt                string  `json:"updated_at"`
}

// Member mirrors the SwitchCloud API project member structure.
type Member struct {
	Id          string `json:"id"`
	ProjectId   string `json:"project_id"`
	UserId      string `json:"user_id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// memberResponse is the shape the provider expects from the API.
type memberResponse struct {
	Id        string             `json:"id"`
	ProjectId string             `json:"project_id"`
	UserId    string             `json:"user_id"`
	CreatedAt string             `json:"created_at"`
	UpdatedAt string             `json:"updated_at"`
	User      memberResponseUser `json:"user"`
}

type memberResponseUser struct {
	Id          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

// overrideEntry stores a route override registered by test code.
type overrideEntry struct {
	method  string
	pattern string
	handler http.HandlerFunc
}

// Server is an in-process HTTP server simulating the SwitchCloud API.
// Use New() to create an instance; call Close() when the test finishes.
type Server struct {
	srv      *httptest.Server
	mux      *http.ServeMux
	mu       sync.RWMutex
	projects map[string]Project
	members  map[string]Member
	orgID    string
	// overrides are checked before the default mux; first match wins.
	overrides []overrideEntry
}

// New creates a fresh Server with empty state and starts it.
func New() *Server {
	s := &Server{
		projects: make(map[string]Project),
		members:  make(map[string]Member),
		orgID:    generateUUID(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/projects", s.handlePostProject)
	mux.HandleFunc("GET /api/v1/projects/{id}", s.handleGetProject)
	mux.HandleFunc("PATCH /api/v1/projects/{id}", s.handlePatchProject)
	mux.HandleFunc("PATCH /api/v1/projects/{id}/archive", s.handleArchiveProject)
	mux.HandleFunc("POST /api/v1/projects/{project_id}/members", s.handlePostMember)
	mux.HandleFunc("GET /api/v1/projects/{project_id}/members/{id}", s.handleGetMember)
	mux.HandleFunc("DELETE /api/v1/projects/{project_id}/members/{id}", s.handleDeleteMember)
	s.mux = mux

	s.srv = httptest.NewServer(http.HandlerFunc(s.ServeHTTP))
	return s
}

// URL returns the base URL of the test server (e.g. "http://127.0.0.1:PORT").
func (s *Server) URL() string { return s.srv.URL }

// Close shuts down the test server.
func (s *Server) Close() { s.srv.Close() }

// Reset clears all overrides and in-memory state, but keeps the server running
// and preserves the organisation ID. Useful when sharing a server across test steps.
func (s *Server) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projects = make(map[string]Project)
	s.members = make(map[string]Member)
	s.overrides = nil
}

// SeedProject pre-populates the server with a known project.
// If the project's TenantId is empty it is set to the server's org ID.
func (s *Server) SeedProject(p Project) *Server {
	if p.TenantId == "" {
		p.TenantId = s.orgID
	}
	if p.CreatedAt == "" {
		p.CreatedAt = time.Now().Format(time.RFC3339)
	}
	if p.UpdatedAt == "" {
		p.UpdatedAt = time.Now().Format(time.RFC3339)
	}
	s.mu.Lock()
	s.projects[p.Id] = p
	s.mu.Unlock()
	return s
}

// SeedMember pre-populates the server with a known project member.
func (s *Server) SeedMember(m Member) *Server {
	if m.CreatedAt == "" {
		m.CreatedAt = time.Now().Format(time.RFC3339)
	}
	if m.UpdatedAt == "" {
		m.UpdatedAt = time.Now().Format(time.RFC3339)
	}
	s.mu.Lock()
	s.members[m.Id] = m
	s.mu.Unlock()
	return s
}

// Override registers a handler that takes precedence over the default logic
// for any request where method and URL path match the given pattern.
// pattern may be an exact path ("/api/v1/projects/abc") or use {wildcard}
// segments ("/api/v1/projects/{id}") which match any non-empty path segment.
// The last registered matching override wins (LIFO search order).
func (s *Server) Override(method, pattern string, handler http.HandlerFunc) *Server {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.overrides = append(s.overrides, overrideEntry{method: method, pattern: pattern, handler: handler})
	return s
}

// RespondWith is a convenience constructor for a handler that always returns
// the given HTTP status code and JSON body string.
func RespondWith(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}
}

// ServeHTTP implements http.Handler. It checks registered overrides before
// falling through to the default stateful route handlers.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	overrides := make([]overrideEntry, len(s.overrides))
	copy(overrides, s.overrides)
	s.mu.RUnlock()

	// Search in reverse so the most recently registered override wins.
	for i := len(overrides) - 1; i >= 0; i-- {
		ov := overrides[i]
		if ov.method == r.Method && matchPattern(ov.pattern, r.URL.Path) {
			ov.handler(w, r)
			return
		}
	}
	s.mux.ServeHTTP(w, r)
}

// matchPattern returns true when the URL path matches the route pattern.
// Segments wrapped in braces (e.g. {id}) match any non-empty path segment.
func matchPattern(pattern, path string) bool {
	pp := strings.Split(strings.Trim(pattern, "/"), "/")
	rp := strings.Split(strings.Trim(path, "/"), "/")
	if len(pp) != len(rp) {
		return false
	}
	for i, seg := range pp {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			if rp[i] == "" {
				return false
			}
			continue
		}
		if seg != rp[i] {
			return false
		}
	}
	return true
}

// ---- Default route handlers ----

func (s *Server) handlePostProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Project struct {
			Name                     string  `json:"name"`
			Description              *string `json:"description,omitempty"`
			CustomerBillingReference *string `json:"customer_billing_reference,omitempty"`
		} `json:"project"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	p := Project{
		Id:                       generateUUID(),
		Name:                     req.Project.Name,
		Description:              req.Project.Description,
		CustomerBillingReference: req.Project.CustomerBillingReference,
		TenantId:                 s.orgID,
		Archived:                 false,
		CreatedAt:                time.Now().Format(time.RFC3339),
		UpdatedAt:                time.Now().Format(time.RFC3339),
	}

	s.mu.Lock()
	s.projects[p.Id] = p
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(p)
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	s.mu.RLock()
	p, ok := s.projects[id]
	s.mu.RUnlock()

	if !ok {
		http.Error(w, `{"error":"project not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(p)
}

func (s *Server) handlePatchProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req struct {
		Project struct {
			Name                     string  `json:"name"`
			Description              *string `json:"description,omitempty"`
			CustomerBillingReference *string `json:"customer_billing_reference,omitempty"`
		} `json:"project"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	p, ok := s.projects[id]
	if !ok {
		s.mu.Unlock()
		http.Error(w, `{"error":"project not found"}`, http.StatusNotFound)
		return
	}

	if req.Project.Name != "" {
		p.Name = req.Project.Name
	}
	p.Description = req.Project.Description
	p.CustomerBillingReference = req.Project.CustomerBillingReference
	p.UpdatedAt = time.Now().Format(time.RFC3339)
	s.projects[id] = p
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(p)
}

func (s *Server) handleArchiveProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	s.mu.Lock()
	p, ok := s.projects[id]
	if !ok {
		s.mu.Unlock()
		http.Error(w, `{"error":"project not found"}`, http.StatusNotFound)
		return
	}
	now := time.Now().Format(time.RFC3339)
	p.Archived = true
	p.ArchivedAt = now
	p.UpdatedAt = now
	s.projects[id] = p
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(p)
}

func (s *Server) handlePostMember(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project_id")

	s.mu.RLock()
	_, projectExists := s.projects[projectID]
	s.mu.RUnlock()

	if !projectExists {
		http.Error(w, `{"error":"project not found"}`, http.StatusNotFound)
		return
	}

	var req struct {
		UserId string `json:"user_id,omitempty"`
		Email  string `json:"email,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	m := Member{
		Id:          generateUUID(),
		ProjectId:   projectID,
		DisplayName: "Test User",
		CreatedAt:   time.Now().Format(time.RFC3339),
		UpdatedAt:   time.Now().Format(time.RFC3339),
	}
	if req.UserId != "" {
		m.UserId = req.UserId
		m.Email = req.UserId + "@example.com"
	} else {
		m.UserId = generateUUID()
		m.Email = req.Email
	}

	s.mu.Lock()
	s.members[m.Id] = m
	s.mu.Unlock()

	resp := toMemberResponse(m)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleGetMember(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project_id")
	id := r.PathValue("id")

	s.mu.RLock()
	_, projectExists := s.projects[projectID]
	m, memberExists := s.members[id]
	s.mu.RUnlock()

	if !projectExists {
		http.Error(w, `{"error":"project not found"}`, http.StatusNotFound)
		return
	}
	if !memberExists || m.ProjectId != projectID {
		http.Error(w, `{"error":"member not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toMemberResponse(m))
}

func (s *Server) handleDeleteMember(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project_id")
	id := r.PathValue("id")

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.projects[projectID]; !ok {
		http.Error(w, `{"error":"project not found"}`, http.StatusNotFound)
		return
	}
	m, ok := s.members[id]
	if !ok || m.ProjectId != projectID {
		http.Error(w, `{"error":"member not found"}`, http.StatusNotFound)
		return
	}

	delete(s.members, id)
	w.WriteHeader(http.StatusNoContent)
}

// ---- Helpers ----

func toMemberResponse(m Member) memberResponse {
	return memberResponse{
		Id:        m.Id,
		ProjectId: m.ProjectId,
		UserId:    m.UserId,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		User: memberResponseUser{
			Id:          m.UserId,
			Email:       m.Email,
			DisplayName: m.DisplayName,
		},
	}
}

func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
