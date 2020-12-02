package auth

import (
	"encoding/json"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/sets"
)

const (
	// kubernetes side
	ClusterRoleKind          = "ClusterRole"
	AdminRole                = "admin"
	ClusterAdminRole         = "cluster-admin"
	UserRolebindingPrefix    = "keystone:user"
	AdminRolebindingPrefix   = "keystone:admin"
	YunionResourceLimitRange = "yunion-limit-range"

	// yunioncloud side
	MemberKeystoneRole       = "_member_"
	AdminKeystoneRole        = "admin"
	SystemKeystoneRole       = "system"
	SystemProject            = "system"
	ProjectOwnerKeystoneRole = "project_owner"
	RoleAssignmentManager    = "role_assignments"
)

var (
	AdminUserRoleSets  sets.String = sets.NewString(AdminKeystoneRole, SystemKeystoneRole)
	NormalUserRoleSets sets.String = sets.NewString(MemberKeystoneRole, ProjectOwnerKeystoneRole)
)

type Meta struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Role Meta

type Project struct {
	Domain Meta `json:"domain"`
	Meta
}

type User struct {
	Domain Meta `json:"domain"`
	Meta
}

type Scope struct {
	Project `json:"project"`
}

type RoleAssignment struct {
	Role  `json:"role"`
	Scope `json:"scope"`
	User  `json:"user"`
}

func NewRoleAssignmentByJSON(obj jsonutils.JSONObject) (r RoleAssignment, err error) {
	r = RoleAssignment{}
	err = obj.Unmarshal(&r)
	return
}

type RoleAssignments []RoleAssignment

func NewRoleAssignmentsByJSON(obj jsonutils.JSONObject) (r RoleAssignments, err error) {
	r = make([]RoleAssignment, 0)
	err = json.Unmarshal([]byte(obj.String()), &r)
	return
}

func (r RoleAssignments) GetFields(getAttr func(RoleAssignment) string) sets.String {
	ret := sets.NewString()
	for _, ra := range r {
		ret.Insert(getAttr(ra))
	}
	return ret
}

func (r RoleAssignments) Projects() sets.String {
	return r.GetFields(func(r RoleAssignment) string {
		return r.Scope.Project.Name
	})
}

func (r RoleAssignments) GetStringFields(getAttr func(RoleAssignment) string) []string {
	ret := r.GetFields(getAttr).Difference(sets.NewString(""))
	return ret.List()
}

func (r RoleAssignments) GetStringField(getAttr func(RoleAssignment) string) string {
	ret := r.GetStringFields(getAttr)
	if len(ret) == 0 {
		return ""
	}
	return ret[0]
}

func (r RoleAssignments) ProjectIDs() []string {
	ret := []string{}
	projects := r.Projects().List()
	for _, project := range projects {
		ret = append(ret, r.GetFields(func(r RoleAssignment) string {
			if r.Scope.Project.Name == project {
				return r.Scope.Project.ID
			}
			return ""
		}).Difference(sets.NewString("")).List()...)
	}
	return ret
}

func (r RoleAssignments) RolesInProject(project string) []string {
	return r.GetStringFields(func(r RoleAssignment) string {
		if r.Scope.Project.Name == project {
			return r.Role.Name
		}
		return ""
	})
}

func (r RoleAssignments) UserID(user string) string {
	return r.GetStringField(func(r RoleAssignment) string {
		if r.User.Name == user {
			return r.User.ID
		}
		return ""
	})
}

func (r RoleAssignments) UserDomain(userIdent string) string {
	return r.GetStringField(func(r RoleAssignment) string {
		if r.User.Name == userIdent || r.User.ID == userIdent {
			return r.Domain.Name
		}
		return ""
	})
}

func (r RoleAssignments) IsAdminUser(userIdent string) (ret bool) {
	for _, ra := range r {
		if ra.User.Name == userIdent || ra.User.ID == userIdent {
			if AdminUserRoleSets.Has(ra.Role.Name) {
				return true
			}
		}
	}
	return false
}

func (r RoleAssignments) UserProjectRole(projectIdent, userIdent string) string {
	for _, ra := range r {
		if ra.User.Name == userIdent || ra.User.ID == userIdent {
			if ra.Scope.Project.Name == projectIdent || ra.Scope.Project.ID == projectIdent {
				return ra.Role.Name
			}
		}
	}
	return ""
}
