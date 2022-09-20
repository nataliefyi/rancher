package charts

type ConstraintYaml struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

type Metadata struct {
	Name string `yaml:"name"`
}

type Spec struct {
	EnforcementAction string     `yaml:"enforcementAction"`
	Match             Match      `yaml:"match"`
	Parameters        Parameters `yaml:"parameters"`
}

type Match struct {
	ExcludedNamespaces []string `yaml:"excludedNamespaces"`
	Kinds              Kinds    `yaml:"kinds"`
}

type Kinds []struct {
	ApiGroups []string `yaml:"apiGroups"`
	Kinds     []string `yaml:"kinds"`
}

type Parameters struct {
	Namespaces []string `yaml:"namespaces"`
}
