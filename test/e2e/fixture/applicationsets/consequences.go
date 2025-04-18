package applicationsets

import (
	"context"
	"encoding/json"
	"time"

	"github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v3/test/e2e/fixture"
	"github.com/argoproj/argo-cd/v3/test/e2e/fixture/applicationsets/utils"
	"github.com/argoproj/argo-cd/v3/util/errors"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
)

// this implements the "then" part of given/when/then
type Consequences struct {
	context *Context
	actions *Actions
}

func (c *Consequences) Expect(e Expectation) *Consequences {
	return c.ExpectWithDuration(e, time.Duration(30)*time.Second)
}

func (c *Consequences) ExpectWithDuration(e Expectation, timeout time.Duration) *Consequences {
	// this invocation makes sure this func is not reported as the cause of the failure - we are a "test helper"
	c.context.t.Helper()
	var message string
	var state state
	sleepIntervals := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		50 * time.Millisecond,
		100 * time.Millisecond,
		200 * time.Millisecond,
		300 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
	}
	sleepIntervalsIdx := -1
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(sleepIntervals[sleepIntervalsIdx]) {
		if sleepIntervalsIdx < len(sleepIntervals)-1 {
			sleepIntervalsIdx++
		}
		state, message = e(c)
		switch state {
		case succeeded:
			log.Infof("expectation succeeded: %s", message)
			return c
		case failed:
			c.context.t.Fatalf("failed expectation: %s", message)
			return c
		}
		log.Infof("expectation pending: %s", message)
	}
	c.context.t.Fatal("timeout waiting for: " + message)
	return c
}

func (c *Consequences) And(block func()) *Consequences {
	c.context.t.Helper()
	block()
	return c
}

func (c *Consequences) Given() *Context {
	return c.context
}

func (c *Consequences) When() *Actions {
	time.Sleep(fixture.WhenThenSleepInterval)
	return c.actions
}

func (c *Consequences) app(name string) *v1alpha1.Application {
	apps := c.apps()

	for index, app := range apps {
		if app.Name == name {
			return &apps[index]
		}
	}

	return nil
}

func (c *Consequences) apps() []v1alpha1.Application {
	c.context.t.Helper()
	var namespace string
	if c.context.switchToNamespace != "" {
		namespace = string(c.context.switchToNamespace)
	} else {
		namespace = fixture.TestNamespace()
	}

	fixtureClient := utils.GetE2EFixtureK8sClient(c.context.t)
	list, err := fixtureClient.AppClientset.ArgoprojV1alpha1().Applications(namespace).List(context.Background(), metav1.ListOptions{})
	errors.CheckError(err)

	if list == nil {
		list = &v1alpha1.ApplicationList{}
	}

	return list.Items
}

func (c *Consequences) applicationSet(applicationSetName string) *v1alpha1.ApplicationSet {
	c.context.t.Helper()
	fixtureClient := utils.GetE2EFixtureK8sClient(c.context.t)

	var appSetClientSet dynamic.ResourceInterface

	if c.context.switchToNamespace != "" {
		appSetClientSet = fixtureClient.ExternalAppSetClientsets[c.context.switchToNamespace]
	} else {
		appSetClientSet = fixtureClient.AppSetClientset
	}

	list, err := appSetClientSet.Get(context.Background(), c.actions.context.name, metav1.GetOptions{})
	errors.CheckError(err)

	var appSet v1alpha1.ApplicationSet

	bytes, err := list.MarshalJSON()
	if err != nil {
		return &v1alpha1.ApplicationSet{}
	}

	err = json.Unmarshal(bytes, &appSet)
	if err != nil {
		return &v1alpha1.ApplicationSet{}
	}

	if appSet.Name == applicationSetName {
		return &appSet
	}

	return &v1alpha1.ApplicationSet{}
}
