package argowf_test

import (
	"testing"

	"github.com/sktelecom/tks-cluster-lcm/pkg/argowf"
)

func TestGetWorkflowTemplates(t *testing.T) {
	cli, err := argowf.New("192.168.97.69", 30004, false, "")
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
	cli, err := argowf.New("192.168.97.69", 30004, false, "")
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
