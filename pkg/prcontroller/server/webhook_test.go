package server_test

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	discoveryfake "k8s.io/client-go/discovery/fake"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/garethjevans/pr-controller/pkg/prcontroller/handler"
	"github.com/garethjevans/pr-controller/pkg/prcontroller/server"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func TestGitHubRequest(t *testing.T) {
	b, err := os.ReadFile("testdata/pr_opened.json")
	if err != nil {
		t.Fatal(err)
	}
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("POST", "/github", strings.NewReader(string(b)))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add("X-GitHub-Delivery", "123456")
	req.Header.Add("X-GitHub-Event", "pull_request")
	req.Header.Add("Content-Type", "application/json")

	h, err := server.NewWebHook("github")
	if err != nil {
		t.Fatal(err)
	}

	exampleGVR := schema.GroupVersionResource{
		Group:    "example.com",
		Version:  "v1alpha1",
		Resource: "examples",
	}

	examplePullRequestGVR := schema.GroupVersionResource{
		Group:    "example.com",
		Version:  "v1alpha1",
		Resource: "examplepullrequests",
	}

	handler.Dynamic = dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			exampleGVR:            "ExampleList",
			examplePullRequestGVR: "ExamplePullRequestList",
		},
		&unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "example.com/v1alpha1",
			"kind":       "Example",
			"metadata": map[string]interface{}{
				"name":      "go-scm",
				"namespace": "my-namespace",
			},
			"spec": map[string]interface{}{
				"source": map[string]interface{}{
					"git": map[string]interface{}{
						"url":    "https://github.com/jenkins-x/go-scm",
						"branch": "main",
					},
				},
			},
		}},
	)

	client := kubernetesfake.NewSimpleClientset()
	fakeDiscovery, ok := client.Discovery().(*discoveryfake.FakeDiscovery)
	if !ok {
		t.Fatalf("couldn't convert Discovery() to *FakeDiscovery")
	}
	fakeDiscovery.Resources = []*v1.APIResourceList{
		{
			TypeMeta:     v1.TypeMeta{},
			GroupVersion: "example.com/v1alpha1",
			APIResources: []v1.APIResource{
				{
					Name:         "examples",
					SingularName: "example",
					Namespaced:   true,
					Group:        "example.com",
					Version:      "v1alpha1",
					Kind:         "Example",
					Categories:   []string{"all-workloads"},
				},
				{
					Name:         "examplepullrequests",
					SingularName: "examplepullrequest",
					Namespaced:   true,
					Group:        "example.com",
					Version:      "v1alpha1",
					Kind:         "ExamplePullRequest",
					Categories:   []string{"all-workloads"},
				},
				{
					Name:         "others",
					SingularName: "other",
					Namespaced:   true,
					Group:        "example.com",
					Version:      "v1alpha1",
					Kind:         "Other",
					Categories:   []string{},
				},
			},
		},
	}

	handler.Discovery = fakeDiscovery

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	requestHandler := http.HandlerFunc(h.Handle)
	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	requestHandler.ServeHTTP(rr, req)
	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusAccepted {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	// Check the response body is what we expect.
	expected := `PR Accepted`
	if strings.TrimSpace(rr.Body.String()) != expected {
		t.Errorf("handler returned unexpected body: got '%v' want '%v'",
			strings.TrimSpace(rr.Body.String()), expected)
	}
}
