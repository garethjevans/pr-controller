package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/client-go/restmapper"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
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
			responseHTTPError(w, 500, fmt.Sprintf("Unable to load config: %v", err))
			return
		}

		Dynamic, err = dynamic.NewForConfig(config)
		if err != nil {
			responseHTTPError(w, 500, fmt.Sprintf("Unable to get dynamic client: %v", err))
			return
		}

		Discovery, err = discovery.NewDiscoveryClientForConfig(config)
		if err != nil {
			responseHTTPError(w, 500, fmt.Sprintf("Unable to get discovery client: %v", err))
			return
		}
	}

	expander := restmapper.NewDiscoveryCategoryExpander(Discovery)
	grs, _ := expander.Expand("all-workloads")

	logrus.Infof("Got %s for all-workloads", grs)

	if len(grs) == 0 {
		responseHTTPError(w, 400, fmt.Sprintf("unable to locate category all-workloads"))
		return
	}

	// we need to locate all types that have a corresponding *PullRequest type
	mappedGrs := toMap(grs)

	for k, v := range mappedGrs {
		logrus.Infof("%s -> %s", k, v)

		mainBranchResources, err := Dynamic.Resource(k.WithVersion("v1alpha1")).List(context.Background(), v1.ListOptions{
			LabelSelector: "",
		})
		if err != nil {
			responseHTTPError(w, 500, fmt.Sprintf("%v", err))
			return
		}

		logrus.Infof("Found %d resources for %s", len(mainBranchResources.Items), k)

		for _, mainBranchResource := range mainBranchResources.Items {
			// we assume that the source is structured how we think...
			gitURL, _, _ := unstructured.NestedString(mainBranchResource.Object, "spec", "source", "git", "url")

			// if the gitURL match
			if strings.TrimSuffix(pr.Repo.Clone, ".git") == strings.TrimSuffix(gitURL, ".git") {
				logrus.Infof("Found matching %s for url %s", mainBranchResources.GetKind(), gitURL)
				if v != nil {
					u := convertToPullRequestType(mainBranchResource, pr)
					logrus.Infof("Creating new resource: %+v\n", u)
					create, err := Dynamic.Resource(v.WithVersion("v1alpha1")).Namespace(u.GetNamespace()).Create(context.Background(), &u, v1.CreateOptions{})
					if err != nil {
						responseHTTPError(w, 500, fmt.Sprintf("%v", err))
						return
					}
					logrus.Infof("Created new resource: %+v\n", create)
				}
			} else {
				logrus.Infof("%s with name %s is a miss", mainBranchResource.GetKind(), mainBranchResource.GetName())
			}
		}
	}

	// FIXME send an accepted response at the end
	responseHTTP(w, http.StatusAccepted, "PR Accepted")
}

func convertToPullRequestType(resource unstructured.Unstructured, pr *scm.PullRequestHook) unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": resource.GetAPIVersion(),
			"kind":       fmt.Sprintf("%sPullRequest", resource.GetKind()),
			"metadata": map[string]interface{}{
				"name":      fmt.Sprintf("%s-pr-%d", resource.GetName(), pr.PullRequest.Number),
				"namespace": resource.GetNamespace(),
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
}

func toMap(grs []schema.GroupResource) map[schema.GroupResource]*schema.GroupResource {
	// FIXME we need to implement this properly
	m := make(map[schema.GroupResource]*schema.GroupResource)
	m[schema.GroupResource{Group: "example.com", Resource: "examples"}] = &schema.GroupResource{Group: "example.com", Resource: "examplepullrequests"}
	return m
}
