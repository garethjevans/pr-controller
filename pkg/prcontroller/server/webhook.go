package server

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/garethjevans/pr-controller/pkg/prcontroller/handler"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/sirupsen/logrus"
)

type webhook struct {
	driver string
	wh     scm.WebhookService
}

type WebHook interface {
	Handle(w http.ResponseWriter, req *http.Request)
}

func NewWebHook(driver string) (WebHook, error) {
	wh, err := factory.NewWebHookService(driver)
	if err != nil {
		return nil, err
	}

	w := &webhook{wh: wh, driver: driver}

	logrus.Infof("Starting Handler for %s", driver)
	logrus.Infof("%s secret is: %s", w.EnvVar(), os.Getenv(w.EnvVar()))

	return w, nil
}

func (w *webhook) EnvVar() string {
	return strings.ToUpper(w.driver) + "_SHARED_SECRET"
}

func (w *webhook) Handle(wr http.ResponseWriter, req *http.Request) {
	logrus.Debugf("handling request... %+v", req)

	hook, err := w.wh.Parse(req, func(webhook scm.Webhook) (string, error) {
		return os.Getenv(w.EnvVar()), nil
	})
	if err != nil {
		handler.ResponseHTTPError(wr, 400, fmt.Sprintf("unable to parse webhook event: %v", err))
		return
	}

	switch hook.Kind() {
	case scm.WebhookKindPullRequest:
		prHook, ok := hook.(*scm.PullRequestHook)
		if ok {
			handler.PullRequest(prHook, wr)
			return
		}
	default:
		logrus.Infof("Unhandled webhook '%s'", hook.Kind())
		handler.ResponseHTTPError(wr, 400, fmt.Sprintf("Unhandled webhook '%s'", hook.Kind()))
		return
	}

	handler.ResponseHTTP(wr, http.StatusAccepted, "Webhook Accepted")
}
