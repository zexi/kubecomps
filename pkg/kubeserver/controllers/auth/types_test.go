package auth

import (
	"encoding/json"
	"reflect"
	"testing"

	"yunion.io/x/pkg/util/sets"
)

var testData = `
[
  {
    "links": {
      "assignment": "http://192.168.0.246:35357/v3/projects/5f6ffb2ca2cf4789957c2eaf8f991c75/users/d453c631368579bdfa34106e912f7d665809189a5a3d0893f55f125f814b5f20/roles/9fe2ff9ee4384b1894a90878d3e92bab"
    },
    "role": {
      "id": "9fe2ff9ee4384b1894a90878d3e92bab",
      "name": "_member_"
    },
    "scope": {
      "project": {
        "domain": {
          "id": "default",
          "name": "Default"
        },
        "id": "5f6ffb2ca2cf4789957c2eaf8f991c75",
        "name": "taikangpoc"
      }
    },
    "user": {
      "domain": {
        "id": "ea4505e018ee44eba8bd828ec66251da",
        "name": "LDAP"
      },
      "id": "d453c631368579bdfa34106e912f7d665809189a5a3d0893f55f125f814b5f20",
      "name": "lizexi"
    }
  },
  {
    "links": {
      "assignment": "http://192.168.0.246:35357/v3/projects/6c77fb028fa849fd8bcd80a2ed42267b/users/d453c631368579bdfa34106e912f7d665809189a5a3d0893f55f125f814b5f20/roles/42d8fda757c14b30a92a11ce10e03294"
    },
    "role": {
      "id": "42d8fda757c14b30a92a11ce10e03294",
      "name": "project_owner"
    },
    "scope": {
      "project": {
        "domain": {
          "id": "default",
          "name": "Default"
        },
        "id": "6c77fb028fa849fd8bcd80a2ed42267b",
        "name": "lizexi"
      }
    },
    "user": {
      "domain": {
        "id": "ea4505e018ee44eba8bd828ec66251da",
        "name": "LDAP"
      },
      "id": "d453c631368579bdfa34106e912f7d665809189a5a3d0893f55f125f814b5f20",
      "name": "lizexi"
    }
  }
]
`

var roleAssignments RoleAssignments

func init() {
	roleAssignments = []RoleAssignment{}
	err := json.Unmarshal([]byte(testData), &roleAssignments)
	if err != nil {
		panic(err)
	}
}

func TestRoleAssignments_Projects(t *testing.T) {
	tests := []struct {
		name string
		r    RoleAssignments
		want sets.String
	}{
		{
			name: "GetAllProjects",
			r:    roleAssignments,
			want: sets.NewString("lizexi", "taikangpoc"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.Projects(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RoleAssignments.Projects() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoleAssignments_RoleInProject(t *testing.T) {
	type args struct {
		project string
	}
	tests := []struct {
		name string
		r    RoleAssignments
		args args
		want string
	}{
		{
			name: "GetProjectOwnerRole",
			r:    roleAssignments,
			args: args{"lizexi"},
			want: ProjectOwnerKeystoneRole,
		},
		{
			name: "GetProjectMemberRole",
			r:    roleAssignments,
			args: args{"taikangpoc"},
			want: MemberKeystoneRole,
		},
		{
			name: "GetNoneRole",
			r:    roleAssignments,
			args: args{"not_exists"},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.RoleInProject(tt.args.project); got != tt.want {
				t.Errorf("RoleAssignments.RoleInProject() = %v, want %v", got, tt.want)
			}
		})
	}
}
