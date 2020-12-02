package auth

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	ttlcache "yunion.io/x/pkg/util/cache"

	o "yunion.io/x/kubecomps/pkg/kubeserver/options"
)

const (
	defaultTimeout          = 600
	roleAssignmentsCacheTTL = 1 * time.Hour
)

type KeystoneAuthenticator struct {
	reconciler           *Reconciler
	roleAssignmentsCache ttlcache.Store
}

// roleAssignmentKeyIndex index RoleAssignments object by userId
func roleAssignmentKeyIndex(obj interface{}) (string, error) {
	ras := obj.(RoleAssignments)
	if len(ras) == 0 {
		return "", fmt.Errorf("Empty RoleAssignments")
	}
	return ras[0].User.ID, nil
}

// NewKeystoneAuthenticator returns a password authenticator that validates credentials using keystone
func NewKeystoneAuthenticator(k8sCli kubernetes.Interface, stopCh chan struct{}) *KeystoneAuthenticator {
	k := &KeystoneAuthenticator{
		roleAssignmentsCache: ttlcache.NewTTLStore(
			roleAssignmentKeyIndex,
			roleAssignmentsCacheTTL),
	}
	reconciler := NewReconciler(k, k8sCli)
	k.reconciler = reconciler

	return k
}

func (k *KeystoneAuthenticator) GetRoleAssignments(userId string) (RoleAssignments, error) {
	ras, err := k.getRoleAssignmentsFromCache(userId)
	if err != nil {
		ras, err = getRoleAssignmentsFromKeystone(userId)
		if err != nil {
			err = fmt.Errorf("Get user %q RoleAssignments from keystone error: %v", userId, err)
			return ras, err
		}
		k.updateRoleAssignmentsInCache(ras)
	} else {
		log.V(10).Debugf("Found userId %q RoleAssignments from cache: %#v", userId, ras)
	}
	return ras, nil
}

func (k *KeystoneAuthenticator) updateRoleAssignmentsInCache(ras RoleAssignments) {
	err := k.roleAssignmentsCache.Update(ras)
	if err != nil {
		log.Errorf("Update RoleAssignments %#v in cache error: %v", ras, err)
		return
	}
	log.V(10).Debugf("Update RoleAssignments %#v", ras)
}

func (k *KeystoneAuthenticator) getRoleAssignmentsFromCache(userId string) (RoleAssignments, error) {
	var ret RoleAssignments = make([]RoleAssignment, 0)
	obj, exists, err := k.roleAssignmentsCache.GetByKey(userId)
	if err != nil {
		err = fmt.Errorf("Get user %q RoleAssignments from ttl cache error: %v", userId, err)
		log.Errorf("Get ttl cache: %v", err)
		return ret, err
	}
	if !exists {
		return ret, fmt.Errorf("Not found user %q RoleAssignments in cache", userId)
	}
	return obj.(RoleAssignments), nil
}

func getRoleAssignmentsFromKeystone(userId string) (RoleAssignments, error) {
	var ret RoleAssignments = make([]RoleAssignment, 0)
	s := auth.AdminSession(context.TODO(), o.Options.Region, "", "", "")
	if s == nil {
		return ret, fmt.Errorf("Can not get auth adminSession")
	}
	query := jsonutils.NewDict()
	query.Add(jsonutils.JSONNull, "include_names")
	query.Add(jsonutils.JSONNull, "effective")
	query.Add(jsonutils.NewString(userId), "user.id")
	data, err := modules.RoleAssignments.List(s, query)
	if err != nil {
		err = fmt.Errorf("List RoleAssignments error: %v", err)
		return ret, err
	}
	if len(data.Data) == 0 {
		return ret, errors.Errorf("User %q RoleAssignments is empty", userId)
	}
	ret, err = NewRoleAssignmentsByJSON(jsonutils.NewArray(data.Data...))
	return ret, err
}

// AuthenticateToken checks the token via Keystone call
func (k *KeystoneAuthenticator) AuthenticateToken(token string) (user.Info, bool, error) {
	cred, err := auth.Verify(context.Background(), token)
	if err != nil {
		err = fmt.Errorf("Failed to verify token %q: %v", token, err)
		log.Errorf("%v", err)
		return nil, false, err
	}

	userName := cred.GetUserName()
	userId := cred.GetUserId()

	ras, err := k.reconciler.ReconcileRBAC(userName, userId)
	if err != nil {
		log.Errorf("ReconcileRBAC for user: %s, id: %s, error: %v", userName, userId, err)
		return nil, false, err
	}

	projects := ras.Projects().List()

	extra := map[string][]string{
		"alpha.kubernetes.io/identity/roles":        cred.GetRoles(),
		"alpha.kubernetes.io/identity/project/id":   ras.ProjectIDs(),
		"alpha.kubernetes.io/identity/project/name": projects,
	}
	authenticatedUser := &user.DefaultInfo{
		Name:   userName,
		UID:    userId,
		Groups: []string{cred.GetProjectName()},
		Extra:  extra,
	}
	return authenticatedUser, true, nil
}
