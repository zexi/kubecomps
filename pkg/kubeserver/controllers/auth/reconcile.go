package auth

import (
	"fmt"

	apiv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	yerrors "yunion.io/x/pkg/util/errors"
	"yunion.io/x/pkg/util/workqueue"

	o "yunion.io/x/kubecomps/pkg/kubeserver/options"
	"yunion.io/x/kubecomps/pkg/kubeserver/types"
)

var (
	ParallelizeWorks                   = 4
	DefaultLimitRange apiv1.LimitRange = apiv1.LimitRange{}
)

func init() {
	cpuDefaultQ, err := resource.ParseQuantity("500m")
	if err != nil {
		log.Fatalf("ParseCpuQuantity err: %v", err)
	}
	cpuDefaultReqQ, _ := resource.ParseQuantity("200m")
	cpuMaxQ, _ := resource.ParseQuantity("16")
	cpuMinQ, _ := resource.ParseQuantity("100m")

	memDefaultQ, _ := resource.ParseQuantity("1024Mi")
	memDefaultReqQ, _ := resource.ParseQuantity("64Mi")
	memMaxQ, _ := resource.ParseQuantity("16Gi")
	memMinQ, err := resource.ParseQuantity("64Mi")
	if err != nil {
		log.Fatalf("ParseMemQuantity err: %v", err)
	}

	DefaultLimitRange = apiv1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name: YunionResourceLimitRange,
		},
		Spec: apiv1.LimitRangeSpec{
			Limits: []apiv1.LimitRangeItem{
				apiv1.LimitRangeItem{
					Type: apiv1.LimitTypeContainer,
					Default: apiv1.ResourceList{
						apiv1.ResourceMemory: memDefaultQ,
						apiv1.ResourceCPU:    cpuDefaultQ,
					},
					DefaultRequest: apiv1.ResourceList{
						apiv1.ResourceMemory: memDefaultReqQ,
						apiv1.ResourceCPU:    cpuDefaultReqQ,
					},
					Max: apiv1.ResourceList{
						apiv1.ResourceMemory: memMaxQ,
						apiv1.ResourceCPU:    cpuMaxQ,
					},
					Min: apiv1.ResourceList{
						apiv1.ResourceMemory: memMinQ,
						apiv1.ResourceCPU:    cpuMinQ,
					},
				},
			},
		},
	}
}

type Reconciler struct {
	keystone  *KeystoneAuthenticator
	K8sClient kubernetes.Interface
}

func NewReconciler(keystone *KeystoneAuthenticator, k8sCli kubernetes.Interface) *Reconciler {
	return &Reconciler{
		keystone:  keystone,
		K8sClient: k8sCli,
	}
}

func RoleAssignmentQuery(userID string) jsonutils.JSONObject {
	query := jsonutils.NewDict()
	query.Add(jsonutils.JSONNull, "include_names")
	query.Add(jsonutils.NewString(userID), "user", "id")
	return query
}

func (r *Reconciler) GetRoleAssignments(userID string) (ret RoleAssignments, err error) {
	return r.keystone.GetRoleAssignments(userID)
}

func (r *Reconciler) GetUserProjects(userID string) ([]string, error) {
	ras, err := r.GetRoleAssignments(userID)
	if err != nil {
		return nil, err
	}
	return ras.Projects().List(), nil
}

func (r *Reconciler) GetUserRoles(userID, project string) ([]string, error) {
	ras, err := r.GetRoleAssignments(userID)
	if err != nil {
		return nil, err
	}
	return ras.RolesInProject(project), nil
}

func Parallelize(execF func(string) error, args ...string) error {
	errsChannel := make(chan error, len(args))
	workqueue.Parallelize(ParallelizeWorks, len(args), func(i int) {
		err := execF(args[i])
		if err != nil {
			errsChannel <- err
			return
		}
	})
	if len(errsChannel) > 0 {
		errs := make([]error, 0)
		length := len(errsChannel)
		for ; length > 0; length-- {
			errs = append(errs, <-errsChannel)
		}
		return yerrors.NewAggregate(errs)
	}
	return nil
}

func IsK8sResourceExist(checkF func() (interface{}, error)) (bool, error) {
	_, err := checkF()
	if errors.IsNotFound(err) {
		return false, nil
	}
	if errors.IsAlreadyExists(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *Reconciler) IsNamespaceExist(name string) (bool, error) {
	return IsK8sResourceExist(func() (interface{}, error) {
		return r.K8sClient.CoreV1().Namespaces().Get(name, metav1.GetOptions{})
	})
}

func (r *Reconciler) IsRoleBindingExist(namespace, name string) (bool, error) {
	return IsK8sResourceExist(func() (interface{}, error) {
		return r.K8sClient.RbacV1().RoleBindings(namespace).Get(name, metav1.GetOptions{})
	})
}

func (r *Reconciler) IsClusterRoleBindingExist(name string) (bool, error) {
	return IsK8sResourceExist(func() (interface{}, error) {
		return r.K8sClient.RbacV1().ClusterRoleBindings().Get(name, metav1.GetOptions{})
	})
}

func (r *Reconciler) IsLimitRangeExist(namespace, name string) (bool, error) {
	return IsK8sResourceExist(func() (interface{}, error) {
		return r.K8sClient.CoreV1().LimitRanges(namespace).Get(name, metav1.GetOptions{})
	})
}

func (r *Reconciler) CreateNamespace(name string) error {
	opt := &apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	_, err := r.K8sClient.CoreV1().Namespaces().Create(opt)
	return err
}

func (r *Reconciler) CreateRoleBinding(namespace, rolebindingName, userName string) error {
	opt := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: rolebindingName,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     ClusterRoleKind,
			Name:     AdminRole,
		},
		Subjects: []rbacv1.Subject{
			{

				Kind:     rbacv1.UserKind,
				APIGroup: rbacv1.GroupName,
				Name:     userName,
			},
		},
	}
	_, err := r.K8sClient.RbacV1().RoleBindings(namespace).Create(opt)
	return err
}

func (r *Reconciler) CreateClusterRoleBinding(clusterRolebindingName, userName string) error {
	opt := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRolebindingName,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     ClusterRoleKind,
			Name:     ClusterAdminRole,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     rbacv1.UserKind,
				APIGroup: rbacv1.GroupName,
				Name:     userName,
			},
		},
	}
	_, err := r.K8sClient.RbacV1().ClusterRoleBindings().Create(opt)
	return err
}

func (r *Reconciler) CreateLimitRange(namespace, name string) error {
	l, err := r.K8sClient.CoreV1().LimitRanges(namespace).Create(&DefaultLimitRange)
	log.Debugf("Create namespace %s LimitRange: %#v, name: %s", namespace, l, name)
	return err
}

func EnsureFunc(
	existsF func() (bool, error),
	createF func() error,
) error {
	exists, err := existsF()
	if err != nil {
		return err
	}
	if !exists {
		err = createF()
		if errors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}
	return nil
}

func (r *Reconciler) EnsureNamespace(project string) error {
	return EnsureFunc(
		func() (bool, error) {
			return r.IsNamespaceExist(project)
		},
		func() error {
			return r.CreateNamespace(project)
		})
}

func (r *Reconciler) EnsureRoleBinding(namespace, rolebindingName, userName string) error {
	return EnsureFunc(
		func() (bool, error) {
			return r.IsRoleBindingExist(namespace, rolebindingName)
		},
		func() error {
			return r.CreateRoleBinding(namespace, rolebindingName, userName)
		})
}

func (r *Reconciler) EnsureClusterRoleBinding(clusterRolebindingName, userName string) error {
	return EnsureFunc(
		func() (bool, error) {
			return r.IsClusterRoleBindingExist(clusterRolebindingName)
		},
		func() error {
			return r.CreateClusterRoleBinding(clusterRolebindingName, userName)
		},
	)
}

func (r *Reconciler) EnsureLimitRange(namespace, name string) error {
	return EnsureFunc(
		func() (bool, error) {
			return r.IsLimitRangeExist(namespace, name)
		},
		func() error {
			return r.CreateLimitRange(namespace, name)
		},
	)
}

func (r *Reconciler) EnsureNamespaces(projects ...string) error {
	return Parallelize(func(name string) error {
		return r.EnsureNamespace(name)
	}, projects...)
}

func (r *Reconciler) EnsureUserProjectsRBAC(userName, userID string, ras RoleAssignments, projects ...string) error {
	rbacRoleName := fmt.Sprintf("%s:%s", userName, userID)
	return Parallelize(func(project string) error {
		roles := ras.RolesInProject(project)
		if project == SystemProject && AdminUserRoleSets.HasAny(roles...) {
			name := fmt.Sprintf("%s:%s", AdminRolebindingPrefix, rbacRoleName)
			return r.EnsureClusterRoleBinding(name, userName)
		}
		name := fmt.Sprintf("%s:%s", UserRolebindingPrefix, rbacRoleName)
		return r.EnsureRoleBinding(project, name, userName)
	}, projects...)
}

func (r *Reconciler) EnsureProjectsLimitRange(projects ...string) error {
	return Parallelize(func(project string) error {
		return r.EnsureLimitRange(project, YunionResourceLimitRange)
	}, projects...)
}

func ProjectsTranslate(names []string) []string {
	ret := []string{}
	for _, name := range names {
		validName := types.ConvertProjectToNamespace(name)
		if validName != "" {
			ret = append(ret, validName)
		}
	}
	return ret
}

func (r *Reconciler) ReconcileRBAC(userName, userID string) (ras RoleAssignments, err error) {
	ras, err = r.GetRoleAssignments(userID)
	if err != nil {
		return
	}

	// map keystone Project to k8s Namespace
	projects := ras.Projects().List()
	projects = ProjectsTranslate(projects)
	log.Debugf("Get projects: %#v", projects)

	err = r.EnsureNamespaces(projects...)
	if err != nil {
		err = fmt.Errorf("Map user %s projects %v namespace err: %v", userName, projects, err)
		return
	}

	// map keystone User role to k8s RBAC
	err = r.EnsureUserProjectsRBAC(userName, userID, ras, projects...)
	if err != nil {
		err = fmt.Errorf("Map user %s projects %v k8s RBAC err: %v", userName, projects, err)
		return
	}

	if o.Options.EnableDefaultLimitRange {
		// create LimitRange to each project namespace
		err = r.EnsureProjectsLimitRange(projects...)
		if err != nil {
			err = fmt.Errorf("Create default LimitRange to projects: %v, error: %v", projects, err)
			return
		}
	}

	// TODO: Start a cronjob to clean invalid RBAC ClusterRoleBindings, RoleBindings and Namespace
	// 1. if rolebinding's namespace not exists in user's projects list, delete it
	// 2. if clusterrolebinding exists, but user is no longer an admin, delete clusterrolebinding
	// 3. if namespace not in Projects, delete it

	return
}
