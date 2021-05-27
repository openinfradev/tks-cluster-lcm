package argowf_test

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/sktelecom/tks-cluster-lcm/pkg/argowf"
)

func TestGetWorkflowTemplates(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "http://localhost:30004/api/v1/workflow-templates/argo",
		httpmock.NewStringResponder(200, `{"items":[{"metadata":{"name":"prepare-argocd","namespace":"argo"},"spec":{"arguments":{"parameters":[{"name":"user","value":"argo"}]}}}]}`))

	cli, err := argowf.New("localhost", 30004, false, "")
	if err != nil {
		t.Errorf("an error was unexpected while initializing argowf client %s", err)
	}

	res, err := cli.GetWorkflowTemplates("argo")
	if err != nil {
		t.Errorf("an error aws unexpected from get workflow-templates api: %s", err)
	}

	for i, item := range res.Items {
		t.Logf("%d) workflow template name: %s", i+1, item.Metadata.Name)
	}
}

func TestSubmitWorkflowFromWftpl(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("POST", "http://localhost:30004/api/v1/workflows/argo/submit",
		httpmock.NewStringResponder(200, `{"metadata":{"name":"prepare-argocd-xxxx","namespace":"argo"}}`))

	cli, err := argowf.New("localhost", 30004, false, "")
	if err != nil {
		t.Errorf("an error was unexpected while initializing argowf client %s", err)
	}

	opts := argowf.SubmitOptions{}
	res, err := cli.SumbitWorkflowFromWftpl("prepare-argocd", "argo", opts, nil)
	if err != nil {
		t.Errorf("an error aws unexpected from get workflow-templates api: %s", err)
	}

	t.Logf("new workflow name: %s", res.Metadata.Name)
}
