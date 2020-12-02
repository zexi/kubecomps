module yunion.io/x/kubecomps

go 1.12

require (
	github.com/Masterminds/semver v1.4.2 // indirect
	github.com/Masterminds/semver/v3 v3.0.1
	github.com/Masterminds/sprig v2.16.0+incompatible
	github.com/ceph/go-ceph v0.0.0-20181217221554-e32f9f0f2e94
	github.com/docker/docker v1.4.2-0.20181221150755-2cb26cfe9cbf
	github.com/fsnotify/fsnotify v1.4.7
	github.com/ghodss/yaml v1.0.0
	github.com/go-sql-driver/mysql v1.5.0
	github.com/gofrs/flock v0.7.1
	github.com/golang/protobuf v1.3.2
	github.com/gorilla/mux v1.7.0
	github.com/gorilla/websocket v1.4.0
	github.com/openshift/api v0.0.0-20191213091414-3fbf6bcf78e8
	github.com/openshift/client-go v0.0.0-20191216194936-57f413491e9e
	github.com/pkg/errors v0.9.1
	github.com/projectcalico/libcalico-go v0.0.0-20200318234626-d63b0c8ad26c
	github.com/sirupsen/logrus v1.4.2
	github.com/zexi/golosetup v0.0.0-20181117053200-8c308e8bbf44
	go.etcd.io/etcd v0.5.0-alpha.5.0.20191023171146-3cf2f69b5738
	golang.org/x/net v0.0.0-20200226121028-0de0cce0169b
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
	google.golang.org/grpc v1.26.0
	gopkg.in/yaml.v2 v2.2.8
	helm.sh/helm/v3 v3.0.0
	k8s.io/api v0.17.3
	k8s.io/apimachinery v0.17.3
	k8s.io/apiserver v0.17.0
	k8s.io/cli-runtime v0.17.0
	k8s.io/client-go v9.0.0+incompatible
	k8s.io/cluster-bootstrap v0.17.3
	k8s.io/gengo v0.0.0-20200425085600-19394052f0fa
	k8s.io/helm v2.12.3+incompatible
	k8s.io/kube-proxy v0.0.0
	k8s.io/kubectl v0.0.0
	k8s.io/kubernetes v1.16.0
	k8s.io/utils v0.0.0-20191114200735-6ca3b61696b6
	sigs.k8s.io/controller-runtime v0.1.10
	sigs.k8s.io/yaml v1.1.0
	yunion.io/x/code-generator v0.0.0-20201109104841-9a6ae1b631f7
	yunion.io/x/jsonutils v0.0.0-20201110084044-3e4e1cb49769
	yunion.io/x/log v0.0.0-20200313080802-57a4ce5966b3
	yunion.io/x/onecloud v0.0.0-20201202090837-45dbd119a439
	yunion.io/x/pkg v0.0.0-20201123083159-ca3aea986ff2
	yunion.io/x/sqlchemy v0.0.0-20201116041103-013e56cab959
)

replace (
	github.com/docker/docker => github.com/docker/docker v0.0.0-20190731150326-928381b2215c
	github.com/renstrom/dedent => github.com/lithammer/dedent v1.1.0
	github.com/ugorji/go => github.com/ugorji/go v1.1.2
	helm.sh/helm/v3 => helm.sh/helm/v3 v3.0.0
	k8s.io/api => k8s.io/api v0.17.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.0
	k8s.io/apimachinery => github.com/openshift/kubernetes-apimachinery v0.0.0-20191211181342-5a804e65bdc1
	k8s.io/apiserver => k8s.io/apiserver v0.17.0
	k8s.io/cli-runtime => github.com/openshift/kubernetes-cli-runtime v0.0.0-20191211181810-5b89652d688e
	k8s.io/client-go => github.com/openshift/kubernetes-client-go v0.0.0-20191211181558-5dcabadb2b45
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.0
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.17.0
	k8s.io/code-generator => k8s.io/code-generator v0.17.0
	k8s.io/component-base => k8s.io/component-base v0.17.0
	k8s.io/cri-api => k8s.io/cri-api v0.17.0
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.17.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.17.0
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.17.0
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.17.0
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.17.0
	k8s.io/kubectl => github.com/openshift/kubernetes-kubectl v0.0.0-20200114121535-5e67185ab42c
	k8s.io/kubelet => k8s.io/kubelet v0.17.0

	k8s.io/kubernetes => github.com/openshift/kubernetes v1.17.0-alpha.0.0.20191216151305-079984b0a154
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.17.0
	k8s.io/metrics => k8s.io/metrics v0.17.0
	k8s.io/node-api => k8s.io/node-api v0.17.0
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.17.0
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.17.0
	k8s.io/sample-controller => k8s.io/sample-controller v0.17.0
)
