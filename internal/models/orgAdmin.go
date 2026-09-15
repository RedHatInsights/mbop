package models

type OrgAdmin struct {
	ID         int  `json:"id"`
	IsOrgAdmin bool `json:"is_org_admin"`
}

type OrgAdminResponse map[int]OrgAdmin
