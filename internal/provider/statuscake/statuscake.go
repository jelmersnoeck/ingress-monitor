package statuscake

import (
	"strconv"
	"time"

	"github.com/jelmersnoeck/ingress-monitor/apis/ingressmonitor/v1alpha1"
	"github.com/jelmersnoeck/ingress-monitor/internal/provider"

	"k8s.io/client-go/kubernetes"

	"github.com/DreamItGetIT/statuscake"
)

// Register registers the provider with a certain factory using the FactoryFunc.
func Register(fact provider.FactoryInterface) {
	fact.Register("StatusCake", FactoryFunc)
}

// FactoryFunc is the function which will allow us to create clients on the fly
// which connect to StatusCake.
func FactoryFunc(k8sClient kubernetes.Interface, prov v1alpha1.NamespacedProvider) (provider.Interface, error) {
	auth := statuscake.Auth{}
	cl, err := statuscake.New(auth)
	if err != nil {
		return nil, err
	}

	return &Client{
		cl:     cl.Tests(),
		groups: prov.StatusCake.ContactGroups,
	}, nil
}

type statusCakeClient interface {
	// The client uses Update for both creation and updating.
	Update(*statuscake.Test) (*statuscake.Test, error)
	Delete(int) error
}

// Client is a wrapper around the StatusCake API Client. This wrapper provides a
// mapping from a Provider interface to the actual StatusCake Client.
type Client struct {
	cl     statusCakeClient
	groups []string
}

// Create translates the MonitorTemplateSpec and creates a new instance with
// StatusCake.
func (c *Client) Create(spec v1alpha1.MonitorTemplateSpec) (string, error) {
	translation, err := c.translateSpec(spec)
	if err != nil {
		return "", err
	}

	test, err := c.cl.Update(translation)
	if err != nil {
		return "", err
	}

	return strconv.Itoa(test.TestID), nil
}

// Delete deletes the monitor which is linked to the given ID from StatusCake.
func (c *Client) Delete(id string) error {
	iid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return err
	}

	return c.cl.Delete(int(iid))
}

// Update updates the Monitor linked to the given ID with the new configuration.
func (c *Client) Update(id string, spec v1alpha1.MonitorTemplateSpec) error {
	iid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return err
	}

	translation, err := c.translateSpec(spec)
	if err != nil {
		return err
	}

	translation.TestID = int(iid)
	_, err = c.cl.Update(translation)
	return err
}

// translateSpec does the actual translation from a MonitorTemplateSpec to a
// StatusCake Test.
func (c *Client) translateSpec(spec v1alpha1.MonitorTemplateSpec) (*statuscake.Test, error) {
	scTest := &statuscake.Test{
		WebsiteName:  spec.Name,
		TestType:     spec.Type,
		ContactGroup: c.groups,
	}

	if spec.Timeout != nil {
		tm, err := time.ParseDuration(*spec.Timeout)
		if err != nil {
			return nil, err
		}

		scTest.Timeout = int(tm.Seconds())
	}

	if spec.CheckRate != nil {
		tm, err := time.ParseDuration(*spec.CheckRate)
		if err != nil {
			return nil, err
		}

		scTest.CheckRate = int(tm.Seconds())
	}

	if spec.Confirmations != nil {
		scTest.Confirmation = *spec.Confirmations
	}

	if http := spec.HTTP; http != nil {
		scTest.CustomHeader = http.CustomHeader
		scTest.UserAgent = http.UserAgent
		scTest.WebsiteURL = http.URL
		scTest.FollowRedirect = http.FollowRedirects
		scTest.FindString = spec.HTTP.ShouldContain

		if spec.HTTP.ShouldNotContain != "" {
			scTest.FindString = spec.HTTP.ShouldNotContain
			scTest.DoNotFind = true
		}
	}

	return scTest, nil
}
