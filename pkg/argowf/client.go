package argowf

import (
	"fmt"
	"errors"
	"sync"
	"io"
	"context"

	"github.com/openinfradev/tks-contract/pkg/log"

	apiclient "github.com/argoproj/argo-workflows/v3/pkg/apiclient"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/argoproj/argo-workflows/v3/util"
	"google.golang.org/grpc/codes"

)

type Client struct {
	serviceClient workflowpkg.WorkflowServiceClient
}

func New(host string, port int) (*Client, error) {
	baseUrl := fmt.Sprintf("%s:%d", host, port)
	opts := apiclient.Opts{
		ArgoServerOpts: apiclient.ArgoServerOpts{
			URL:                baseUrl,
			Secure:             false,
			InsecureSkipVerify: false,
			HTTP1: true,
		},
		AuthSupplier: func() string {
			return ""
		},
	}
	_, client, err := apiclient.NewClientFromOpts(opts)
	if err != nil {
		return nil, err
	}
	serviceClient := client.NewWorkflowServiceClient()

	return &Client{
		serviceClient: serviceClient,
	}, nil
}



func (c *Client) SumbitWorkflowFromWftpl(ctx context.Context, wftplName string, namespace string, parameters []string) (string, error) {
	submitOpts := wfv1.SubmitOpts{}
	submitOpts.Parameters = parameters

	created, err := c.serviceClient.SubmitWorkflow(ctx, &workflowpkg.WorkflowSubmitRequest{
		Namespace:     namespace,
		ResourceKind:  "WorkflowTemplate",
		ResourceName:  wftplName,
		SubmitOptions: &submitOpts,
	})

	if( err != nil ){
		log.Error( "Failed to submit : err ", err )
		return "", err
	}

	log.Debug( "SumbitWorkflowFromWftpl created name : ", created.Name )

	return created.Name, nil
}

func (c *Client) IsRunningWorkflowByContractId(ctx context.Context, nameSpace string, contractId string) (bool, error) {
	wfList, err := c.serviceClient.ListWorkflows(ctx, &workflowpkg.WorkflowListRequest{
		Namespace:   nameSpace,
		ListOptions: &metav1.ListOptions{LabelSelector: "workflows.argoproj.io/phase in (Pending,Running)"},
		Fields:      "items.metadata.name,items.spec",
	})

	if err != nil {
		log.Error( "failed to get argo workflows namespace. err : ", err )
		return false, err
	}

	for _, item := range wfList.Items {
		log.Debug(item)
		for _, arg := range item.Spec.Arguments.Parameters {
			if arg.Name == "contract_id" && arg.Value.String() == contractId {
				return true, errors.New("Existed running(pending) workflow")
			}
		}
	}

	return false, nil
}

func (c *Client) WaitWorkflows(ctx context.Context, namespace string, workflowNames []string, ignoreNotFound, quiet bool) bool {
	log.Debug( "waiting workflowNames : ", workflowNames )

	var wg sync.WaitGroup
	wfSuccessStatus := true

	for _, name := range workflowNames {
		wg.Add(1)
		go func(name string) {
			if !c.waitOnOne(ctx, name, namespace, ignoreNotFound, quiet) {
				wfSuccessStatus = false
			}
			wg.Done()
		}(name)

	}
	wg.Wait()

	return wfSuccessStatus
}

func (c *Client) waitOnOne(ctx context.Context, wfName, namespace string, ignoreNotFound, quiet bool) bool {
	req := &workflowpkg.WatchWorkflowsRequest{
		Namespace: namespace,
		ListOptions: &metav1.ListOptions{
			FieldSelector:   util.GenerateFieldSelectorFromWorkflowName(wfName),
			ResourceVersion: "0",
		},
	}
	stream, err := c.serviceClient.WatchWorkflows(ctx, req)
	if err != nil {
		if status.Code(err) == codes.NotFound && ignoreNotFound {
			return true
		}
		return false
	}
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			log.Debug("Re-establishing workflow watch")
			stream, _ = c.serviceClient.WatchWorkflows(ctx, req)
			continue
		}
		if event == nil {
			continue
		}
		wf := event.Object
		if !wf.Status.FinishedAt.IsZero() {
			if !quiet {
				log.Info(fmt.Sprintf("%s %s at %v\n", wfName, wf.Status.Phase, wf.Status.FinishedAt))
			}
			if wf.Status.Phase == wfv1.WorkflowFailed || wf.Status.Phase == wfv1.WorkflowError {
				return false
			}
			return true
		}
	}
}

