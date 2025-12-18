// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-faker/faker/v4"
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
	Id          string `json:"id"`
	ProjectId   string `json:"project_id"`
	UserId      string `json:"user_id"`
	EMail       string `json:"email"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type ProjectMemberResponse struct {
	Id        string                     `json:"id"`
	ProjectId string                     `json:"project_id"`
	UserId    string                     `json:"user_id"`
	CreatedAt string                     `json:"created_at"`
	UpdatedAt string                     `json:"updated_at"`
	Links     ProjectMemberResponseLinks `json:"links"`
	User      ProjectMemberResponseUser  `json:"user"`
}

type ProjectMemberResponseLinks struct {
	Project string `json:"project"`
}

type ProjectMemberResponseUser struct {
	Id          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

var orgId string = faker.UUIDHyphenated()
var projects map[string]Project = make(map[string]Project)
var projectMembers map[string]ProjectMember = make(map[string]ProjectMember)

func handleDebug(w http.ResponseWriter, r *http.Request) {

	type debugResponse struct {
		Projects      map[string]Project       `json:"projects"`
		ProjectMember map[string]ProjectMember `json:"project_members"`
	}

	var response = debugResponse{
		Projects:      projects,
		ProjectMember: projectMembers,
	}

	p, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(p)
}

func handlePostProject(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var p Project
	err := decoder.Decode(&p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	p.Id = faker.UUIDHyphenated()
	p.OrganisationId = orgId
	p.CreatedAt = time.Now().Format(time.RFC3339)
	p.UpdatedAt = time.Now().Format(time.RFC3339)

	projects[p.Id] = p

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Printf("Created Project: %+v\n", p)
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
	fmt.Printf("Get Project: %+v\n", projects[id])
	json.NewEncoder(w).Encode(projects[id])
}

func handlePostProjectMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	project_id := vars["project_id"]

	if _, ok := projects[project_id]; !ok {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	decoder := json.NewDecoder(r.Body)
	var p ProjectMember
	err := decoder.Decode(&p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Printf("Debug Project Member: %+v\n", p)

	p.Id = faker.UUIDHyphenated()
	if p.UserId == "" {
		p.UserId = faker.UUIDHyphenated()
	}
	if p.EMail == "" {
		p.EMail = faker.Email()
	}

	p.DisplayName = faker.Name()
	p.ProjectId = project_id
	p.CreatedAt = time.Now().Format(time.RFC3339)
	p.UpdatedAt = time.Now().Format(time.RFC3339)

	projectMembers[p.Id] = p

	response := ProjectMemberResponse{
		Id:        projectMembers[p.Id].Id,
		ProjectId: projectMembers[p.Id].ProjectId,
		UserId:    projectMembers[p.Id].UserId,
		CreatedAt: projectMembers[p.Id].CreatedAt,
		UpdatedAt: projectMembers[p.Id].UpdatedAt,
		Links: ProjectMemberResponseLinks{
			Project: "/api/v1/projects/" + projectMembers[p.Id].ProjectId,
		},
		User: ProjectMemberResponseUser{
			Id:          projectMembers[p.Id].UserId,
			Email:       projectMembers[p.Id].EMail,
			DisplayName: projectMembers[p.Id].DisplayName,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Printf("Created Project Member: %+v\n", p)
	json.NewEncoder(w).Encode(response)
}

func handleGetProjectMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	project_id := vars["project_id"]
	if _, ok := projects[project_id]; !ok {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	if _, ok := projectMembers[id]; !ok {
		http.Error(w, "Project Member not found", http.StatusNotFound)
		return
	}

	response := ProjectMemberResponse{
		Id:        projectMembers[id].Id,
		ProjectId: projectMembers[id].ProjectId,
		UserId:    projectMembers[id].UserId,
		CreatedAt: projectMembers[id].CreatedAt,
		UpdatedAt: projectMembers[id].UpdatedAt,
		Links: ProjectMemberResponseLinks{
			Project: "/api/v1/projects/" + projectMembers[id].ProjectId,
		},
		User: ProjectMemberResponseUser{
			Id:          projectMembers[id].UserId,
			Email:       projectMembers[id].EMail,
			DisplayName: projectMembers[id].DisplayName,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Printf("Get Project Member: %+v\n", response)
	json.NewEncoder(w).Encode(response)
}

func handleDeleteProjectMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	project_id := vars["project_id"]
	if _, ok := projects[project_id]; !ok {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	if _, ok := projectMembers[id]; !ok {
		http.Error(w, "Project Member not found", http.StatusNotFound)
		return
	}

	delete(projectMembers, id)

	w.WriteHeader(http.StatusNoContent)
	w.Header().Set("Content-Type", "application/json")
	fmt.Printf("Deleted Project Member: %+v\n", id)
}

func main() {

	var p Project = Project{
		Id:             "0faaecfb-d154-4f8f-bdc8-fccd630ddb39",
		Name:           "test1",
		Description:    nil,
		OrganisationId: orgId,
		Archived:       false,
		ArchivedAt:     "",
		CreatedAt:      "2024-01-01T00:00:00Z",
		UpdatedAt:      "2024-01-01T00:00:00Z",
	}
	projects[p.Id] = p

	r := mux.NewRouter()

	r.HandleFunc("/debug", handleDebug).Methods("GET")
	r.HandleFunc("/api/v1/projects", handlePostProject).Methods("POST")
	r.HandleFunc("/api/v1/projects/{id}", handleGetProject).Methods("GET")
	r.HandleFunc("/api/v1/projects/{project_id}/members", handlePostProjectMember).Methods("POST")
	r.HandleFunc("/api/v1/projects/{project_id}/members/{id}", handleGetProjectMember).Methods("GET")
	r.HandleFunc("/api/v1/projects/{project_id}/members/{id}", handleDeleteProjectMember).Methods("DELETE")

	err := http.ListenAndServe(":3000", r)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}
