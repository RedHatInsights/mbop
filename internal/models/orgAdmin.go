package models

type OrgAdmin struct {
	ID         string `json:"id"`
	IsOrgAdmin bool   `json:"is_org_admin"`
}

type OrgAdminResponse map[string]OrgAdmin
