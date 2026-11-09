package models

import (
	"testing"
)

func TestUsers_RemoveNonOrgAdmins(t *testing.T) {
	type fields struct {
		UserCount int
		Users     []User
	}
	tests := []struct {
		name          string
		fields        fields
		expectedUsers []User
		expectedCount int
	}{
		{
			name: "remove non-org-admins from mixed users",
			fields: fields{
				UserCount: 4,
				Users: []User{
					{Username: "admin1", ID: "1", IsOrgAdmin: true},
					{Username: "user1", ID: "2", IsOrgAdmin: false},
					{Username: "admin2", ID: "3", IsOrgAdmin: true},
					{Username: "user2", ID: "4", IsOrgAdmin: false},
				},
			},
			expectedUsers: []User{
				{Username: "admin1", ID: "1", IsOrgAdmin: true},
				{Username: "admin2", ID: "3", IsOrgAdmin: true},
			},
			expectedCount: 2,
		},
		{
			name: "all users are org-admins",
			fields: fields{
				UserCount: 3,
				Users: []User{
					{Username: "admin1", ID: "1", IsOrgAdmin: true},
					{Username: "admin2", ID: "2", IsOrgAdmin: true},
					{Username: "admin3", ID: "3", IsOrgAdmin: true},
				},
			},
			expectedUsers: []User{
				{Username: "admin1", ID: "1", IsOrgAdmin: true},
				{Username: "admin2", ID: "2", IsOrgAdmin: true},
				{Username: "admin3", ID: "3", IsOrgAdmin: true},
			},
			expectedCount: 3,
		},
		{
			name: "no users are org-admins",
			fields: fields{
				UserCount: 2,
				Users: []User{
					{Username: "user1", ID: "1", IsOrgAdmin: false},
					{Username: "user2", ID: "2", IsOrgAdmin: false},
				},
			},
			expectedUsers: []User{},
			expectedCount: 0,
		},
		{
			name: "empty users list",
			fields: fields{
				UserCount: 0,
				Users:     []User{},
			},
			expectedUsers: []User{},
			expectedCount: 0,
		},
		{
			name: "single org-admin user",
			fields: fields{
				UserCount: 1,
				Users: []User{
					{Username: "admin1", ID: "1", IsOrgAdmin: true},
				},
			},
			expectedUsers: []User{
				{Username: "admin1", ID: "1", IsOrgAdmin: true},
			},
			expectedCount: 1,
		},
		{
			name: "single non-org-admin user",
			fields: fields{
				UserCount: 1,
				Users: []User{
					{Username: "user1", ID: "1", IsOrgAdmin: false},
				},
			},
			expectedUsers: []User{},
			expectedCount: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &Users{
				UserCount: tt.fields.UserCount,
				Users:     tt.fields.Users,
			}
			u.RemoveNonOrgAdmins()

			// Check that the correct number of users remain
			if len(u.Users) != tt.expectedCount {
				t.Errorf("RemoveNonOrgAdmins() resulted in %d users, expected %d", len(u.Users), tt.expectedCount)
			}

			// Check that all remaining users are org-admins
			for i, user := range u.Users {
				if !user.IsOrgAdmin {
					t.Errorf("RemoveNonOrgAdmins() user at index %d is not an org-admin: %+v", i, user)
				}
			}

			// Check that the expected users are present
			if len(u.Users) != len(tt.expectedUsers) {
				t.Errorf("RemoveNonOrgAdmins() resulted in %d users, expected %d users", len(u.Users), len(tt.expectedUsers))
				return
			}

			for i, expectedUser := range tt.expectedUsers {
				if i >= len(u.Users) {
					t.Errorf("RemoveNonOrgAdmins() missing expected user at index %d: %+v", i, expectedUser)
					continue
				}
				actualUser := u.Users[i]
				if actualUser.Username != expectedUser.Username || actualUser.ID != expectedUser.ID || actualUser.IsOrgAdmin != expectedUser.IsOrgAdmin {
					t.Errorf("RemoveNonOrgAdmins() user at index %d = %+v, expected %+v", i, actualUser, expectedUser)
				}
			}
		})
	}
}
