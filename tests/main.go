// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

type Project struct {
	Id             string  `json:"id"`
	Name           string  `json:"name"`
	Description    *string `json:"description,omitempty"`
	OrganisationId string  `json:"organisation_id"`
	Archived       bool    `json:"archived"`
	ArchivedAt     string  `json:"archived_at"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

type ProjectMember struct {
	Id        string `json:"id"`
	ProjectId string `json:"project_id"`
	UserId    string `json:"user_id"`
	Role      string `json:"role"`
}

var projects map[string]Project = make(map[string]Project)

func handlePostProject(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var p Project
	err := decoder.Decode(&p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	p.Id = "proj-" + fmt.Sprintf("%d", len(projects)+1)
	p.OrganisationId = "org-123456"
	p.CreatedAt = "2024-01-01T00:00:00Z"
	p.UpdatedAt = "2024-01-01T00:00:00Z"

	projects[p.Id] = p

	json.NewEncoder(os.Stdout).Encode(p)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

func handleGetProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if _, ok := projects[id]; !ok {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects[id])
}

func main() {
	var p Project = Project{
		Id:             "0faaecfb-d154-4f8f-bdc8-fccd630ddb39",
		Name:           "test1",
		Description:    nil,
		OrganisationId: "org-123456",
		Archived:       false,
		ArchivedAt:     "",
		CreatedAt:      "2024-01-01T00:00:00Z",
		UpdatedAt:      "2024-01-01T00:00:00Z",
	}
	projects[p.Id] = p

	r := mux.NewRouter()
	r.HandleFunc("/api/v1/projects", handlePostProject).Methods("POST")
	r.HandleFunc("/api/v1/projects/{id}", handleGetProject).Methods("GET")

	err := http.ListenAndServe(":3000", r)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}
