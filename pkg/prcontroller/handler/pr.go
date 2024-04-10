package handler

import (
	"context"
	"fmt"
	"github.com/garethjevans/pr-controller/pkg/defines"
	"net/http"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

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
	logrus.Infof("handling %s for PR-%d", pr.Action, pr.PullRequest.Number)
	logrus.Debugf("%+v", pr)

	ctx := context.Background()

	if Dynamic == nil {
		// can we locate a workload for this hook?
		config, err := rest.InClusterConfig()
		if err != nil {
			logrus.Errorf("Unable to load config: %v", err)
			ResponseHTTPError(w, 500, fmt.Sprintf("Unable to load config: %v", err))
			return
		}

		Dynamic, err = dynamic.NewForConfig(config)
		if err != nil {
			logrus.Errorf("Unable to get dynamic client: %v", err)
			ResponseHTTPError(w, 500, fmt.Sprintf("Unable to get dynamic client: %v", err))
			return
		}
	}

	supplyChainList, err := Dynamic.Resource(schema.GroupVersionResource{
		Group:    "supply-chain.apps.tanzu.vmware.com",
		Version:  "v1alpha1",
		Resource: "supplychains",
	}).List(ctx, v1.ListOptions{})
	if err != nil {
		logrus.Errorf("Unable to get supply chains: %v", err)
		ResponseHTTPError(w, 500, fmt.Sprintf("Unable to get supply chains: %v", err))
		return
	}

	var kinds []defines.GroupVersionResourceKind
	for _, supplyChain := range supplyChainList.Items {
		kinds = append(kinds, defines.Workload(supplyChain))
	}

	// we need to locate all types that have a corresponding *PullRequest type
	mappedGrs := ToMap(kinds)

	logrus.Debugf("mapped GroupResources %s", mappedGrs)

	logrus.Infof("seaching for resources for git url %s and target branch %s", strings.TrimSuffix(pr.Repo.Clone, ".git"), pr.PullRequest.Target)

	for k, v := range mappedGrs {
		logrus.Infof("%s -> %s", k.Kind, v.Kind)

		mainBranchResources, err := Dynamic.Resource(k.ToGroupVersionResource()).List(context.Background(), v1.ListOptions{
			LabelSelector: "",
		})
		if err != nil {
			ResponseHTTPError(w, 500, fmt.Sprintf("%v", err))
			return
		}

		logrus.Infof("Found %d resources for %s", len(mainBranchResources.Items), k.Kind)

		found := false

		for _, mainBranchResource := range mainBranchResources.Items {
			// we assume that the source is structured how we think...
			gitURL, _, _ := unstructured.NestedString(mainBranchResource.Object, "spec", "source", "git", "url")
			branch, _, _ := unstructured.NestedString(mainBranchResource.Object, "spec", "source", "git", "branch")

			// if the gitURL match
			if strings.TrimSuffix(pr.Repo.Clone, ".git") == strings.TrimSuffix(gitURL, ".git") && pr.PullRequest.Target == branch {
				logrus.Infof("Found matching %s for url %s", mainBranchResources.GetKind(), gitURL)
				found = true
				u := convertToPullRequestType(mainBranchResource, v, pr)

				switch pr.Action.String() {
				case "create", "updated", "opened", "reopened":
					createOrUpdate(Dynamic, u, w, v)
					return
				case "merged", "closed":
					deleteIfExists(Dynamic, u, w, v)
					return
				default:
					logrus.Warnf("unhandled action %s", pr.Action)
				}
			}
		}

		if !found {
			logrus.Infof("couldn't find a matching %s resource for PR-%d", k.Kind, pr.PullRequest.Number)
		}

	}

	ResponseHTTP(w, http.StatusAccepted, "PR Accepted")
}

func deleteIfExists(d dynamic.Interface, u unstructured.Unstructured, w http.ResponseWriter, v defines.GroupVersionResourceKind) {
	logrus.Infof("Delete handler: %s", u.GetName())

	// we should check if this resource already exists
	got, err := d.Resource(v.ToGroupVersionResource()).Namespace(u.GetNamespace()).Get(context.Background(), u.GetName(), v1.GetOptions{})
	if err != nil {
		logrus.Infof("unable to determine if %s exists: %v", u.GetName(), err)
	}

	if got != nil {
		logrus.Infof("Deleting resource: %s\n", u.GetName())
		err := d.Resource(v.ToGroupVersionResource()).Namespace(u.GetNamespace()).Delete(context.Background(), got.GetName(), v1.DeleteOptions{})
		if err != nil {
			logrus.Errorf("unable to delete %s: %v", got.GetName(), err)
			ResponseHTTPError(w, 500, fmt.Sprintf("%v", err))
			return
		}
		logrus.Infof("Deleted resource: %s\n", got.GetName())
	}

	ResponseHTTP(w, http.StatusCreated, "Resource Deleted")
}

func createOrUpdate(d dynamic.Interface, u unstructured.Unstructured, w http.ResponseWriter, v defines.GroupVersionResourceKind) {
	logrus.Infof("CreateOrUpdate handler: %s", u.GetName())

	// we should check if this resource already exists
	got, err := d.Resource(v.ToGroupVersionResource()).Namespace(u.GetNamespace()).Get(context.Background(), u.GetName(), v1.GetOptions{})
	if err != nil {
		logrus.Infof("unable to determine if %s exists: %v", u.GetName(), err)
	}

	if got == nil {
		logrus.Infof("Creating new resource: %+v", u)
		create, err := d.Resource(v.ToGroupVersionResource()).Namespace(u.GetNamespace()).Create(context.Background(), &u, v1.CreateOptions{})
		if err != nil {
			logrus.Errorf("unable to create %s: %v", got.GetName(), err)
			ResponseHTTPError(w, 500, fmt.Sprintf("%v", err))
			return
		}
		logrus.Infof("Created new resource: %s", create.GetName())
	} else {
		logrus.Infof("Updating resource: %s", got.GetName())
		branch, _, _ := unstructured.NestedString(u.UnstructuredContent(), "spec", "source", "git", "branch")
		_ = unstructured.SetNestedField(got.UnstructuredContent(), branch, "spec", "source", "git", "branch")

		commit, _, _ := unstructured.NestedString(u.UnstructuredContent(), "spec", "source", "git", "commit")
		_ = unstructured.SetNestedField(got.UnstructuredContent(), commit, "spec", "source", "git", "commit")

		_, err = d.Resource(v.ToGroupVersionResource()).Namespace(u.GetNamespace()).Update(context.Background(), got, v1.UpdateOptions{})
		if err != nil {
			logrus.Errorf("unable to update %s: %v", got.GetName(), err)
			ResponseHTTPError(w, 500, fmt.Sprintf("%v", err))
			return
		}
	}

	ResponseHTTP(w, http.StatusCreated, "Resource Created")
}

func convertToPullRequestType(resource unstructured.Unstructured, gvrk defines.GroupVersionResourceKind, pr *scm.PullRequestHook) unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": resource.GetAPIVersion(),
			"kind":       gvrk.Kind,
			"metadata": map[string]interface{}{
				"name":      fmt.Sprintf("%s-pr-%d", resource.GetName(), pr.PullRequest.Number),
				"namespace": resource.GetNamespace(),
			},
			"spec": map[string]interface{}{
				"source": map[string]interface{}{
					"git": map[string]interface{}{
						"url":    pr.Repo.Clone,
						"branch": pr.PullRequest.Head.Ref,
						"commit": pr.PullRequest.Sha,
					},
				},
			},
			// TODO at some point we will want to consider some kind of additional mapping here.
			// for example, how do we set extra properties that are required for tests
		},
	}
}

func ToMap(in []defines.GroupVersionResourceKind) map[defines.GroupVersionResourceKind]defines.GroupVersionResourceKind {
	m := make(map[defines.GroupVersionResourceKind]defines.GroupVersionResourceKind)

	for _, i := range in {
		//fmt.Printf("checking %s\n", i)
		if isNotPullRequestResource(i) {
			//fmt.Printf("%s is not a pull request CR\n", i)
			pr := locatePullRequestResourceForBaseResource(i, in)
			if pr != nil {
				m[i] = *pr
			}
		}
	}

	return m
}

func locatePullRequestResourceForBaseResource(base defines.GroupVersionResourceKind, in []defines.GroupVersionResourceKind) *defines.GroupVersionResourceKind {
	for _, i := range in {
		if i.Group == base.Group && !isNotPullRequestResource(i) && matches(base.Resource, i.Resource) {
			return &i
		}
	}
	return nil
}

func matches(base string, resource string) bool {
	return strings.TrimSuffix(base, "s") == strings.TrimSuffix(strings.TrimSuffix(resource, "prs"), "pullrequests")
}

func isNotPullRequestResource(i defines.GroupVersionResourceKind) bool {
	return !strings.HasSuffix(i.Resource, "prs") && !strings.HasSuffix(i.Resource, "pullrequests")
}
