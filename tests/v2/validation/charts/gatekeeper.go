package charts

import "k8s.io/apimachinery/pkg/runtime/schema"

const (
	// Project that charts are installed in
	gatekeeperProjectName = "gatekeeper-project"
)

var Constraint = schema.GroupVersionResource{
	Group:    "constraints.gatekeeper.sh",
	Version:  "v1beta1",
	Resource: "k8srequiredlabels",
}

var Namespaces = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "namespaces",
}
