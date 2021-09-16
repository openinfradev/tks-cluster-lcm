package argowf

// GetWorkflowTemplatesResponse is a response from GET /api/v1/workflow-templates API.
type GetWorkflowTemplatesResponse struct {
	Items []workflowTemplate `json:"items"`
}

type workflowTemplate struct {
	Metadata metadata             `json:"metadata"`
	Spec     workflowTemplateSpec `json:"spec"`
}

type workflowTemplateSpec struct {
	Args arguments `json:"arguments"`
}

type metadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type arguments struct {
	Parameters []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"parameters"`
}

type submitWorkflowRequestBody struct {
	Namespace     string        `json:"namespace,omitempty"`
	ResourceKind  string        `json:"resourceKind,omitempty"`
	ResourceName  string        `json:"resourceName,omitempty"`
	SubmitOptions SubmitOptions `json:"submitOptions,omitempty"`
}

// SubmitOptions is optional fields to submit new workflow.
type SubmitOptions struct {
	DryRun       bool   `json:"dryRun,omitempty"`
	EntryPoint   string `json:"entryPoint,omitempty"`
	GenerateName string `json:"generateName,omitempty"`
	Labels       string `json:"labels,omitempty"`
	Name         string `json:"name,omitempty"`
	Parameters   []string `json:"parameters,omitempty"`
}

type SubmitWorkflowResponse struct {
	Metadata metadata `json:"metadata"`
}

type GetWorkflowsResponse struct {
	Items []workflow `json:"items"`
}

type workflow struct {
	ApiVersion string `json:"apiVersion"`
	Metadata workflowMetadata `json:"metadata"`
	Spec     workflowSpec `json:"spec"`
	Status   workflowStatus `json:"status"`
}

type workflowMetadata struct {
	ClusterName string `json:"clusterName"`
	GenerateName string `json:"generateName"`
	Name string `json:"name"`
	NameSpace string `json:"namespace"`
}

type workflowSpec struct {
	Args workflowArguments `json:"arguments"`
}

type workflowArguments struct {
	Parameters []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"parameters"`
}

type workflowStatus struct {
	Phase string `json:"phase"`
	Message string `json:"message"`
}
