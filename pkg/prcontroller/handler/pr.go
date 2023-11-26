package handler

import (
	"context"
	"fmt"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"net/http"
)

var (
	Dynamic   dynamic.Interface
	Discovery discovery.DiscoveryInterface
)

func PullRequest(pr *scm.PullRequestHook, w http.ResponseWriter) {
	if Dynamic == nil {
		// can we locate a workload for this hook?
		config, err := rest.InClusterConfig()
		if err != nil {
			responseHTTPError(w, 500, fmt.Sprintf("%v", err))
			return
		}

		Dynamic, err = dynamic.NewForConfig(config)
		if err != nil {
			responseHTTPError(w, 500, fmt.Sprintf("%v", err))
			return
		}

		Discovery, err = discovery.NewDiscoveryClientForConfig(config)
		if err != nil {
			responseHTTPError(w, 500, fmt.Sprintf("%v", err))
			return
		}
	}

	gvr := schema.GroupVersionResource{
		Group:    "example.com",
		Version:  "v1alpha1",
		Resource: "examplepullrequests",
	}

	pullRequestResources, err := Dynamic.Resource(gvr).List(context.Background(), v1.ListOptions{
		LabelSelector: "",
	})
	if err != nil {
		// FIXME
	}

	if len(pullRequestResources.Items) == 1 {
		// we should update this with the latest sha
	} else {
		u := unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": fmt.Sprintf("%s/%s", "example.com", "v1alpha1"),
				"kind":       "ExamplePullRequest",
				"metadata": map[string]interface{}{
					"name": fmt.Sprintf("%s-pr-%d", pr.Repo.Name, pr.PullRequest.Number),
				},
				"spec": map[string]interface{}{
					"source": map[string]interface{}{
						"git": map[string]interface{}{
							"url":    pr.Repo.Clone,
							"branch": pr.PullRequest.Base.Ref,
							"commit": pr.PullRequest.Sha,
						},
					},
				},
			},
		}

		logrus.Infof("%+v\n", u)
	}

	// FIXME send an accepted response at the end
	responseHTTP(w, http.StatusAccepted, "PR Accepted")
}
