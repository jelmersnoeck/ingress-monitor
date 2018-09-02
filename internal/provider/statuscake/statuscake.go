package statuscake

import (
	"fmt"
	"strconv"
	"time"

	"github.com/jelmersnoeck/ingress-monitor/apis/ingressmonitor/v1alpha1"
	"github.com/jelmersnoeck/ingress-monitor/internal/provider"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/DreamItGetIT/statuscake"
)

// The StatusCake Client we're using expects us to set this.
// XXX remove this once we move to our own internal client.
const statusCodes = "204,205,206,303,400,401,403,404,405,406,408,410,413,444,429,494,495,496,499,500,501,502,503,504,505,506,507,508,509,510,511,521,522,523,524,520,598,599"

// Register registers the provider with a certain factory using the FactoryFunc.
func Register(fact provider.FactoryInterface) {
	fact.Register("StatusCake", FactoryFunc)
}

// FactoryFunc is the function which will allow us to create clients on the fly
// which connect to StatusCake.
func FactoryFunc(k8sClient kubernetes.Interface, prov v1alpha1.NamespacedProvider) (provider.Interface, error) {
	username, err := getSecretValue(k8sClient, prov.Namespace, prov.StatusCake.Username)
	if err != nil {
		return nil, err
	}

	apiKey, err := getSecretValue(k8sClient, prov.Namespace, prov.StatusCake.APIKey)
	if err != nil {
		return nil, err
	}

	auth := statuscake.Auth{
		Username: username,
		Apikey:   apiKey,
	}
	cl, err := statuscake.New(auth)
	if err != nil {
		return nil, err
	}

	return &Client{
		cl:     cl.Tests(),
		groups: prov.StatusCake.ContactGroups,
	}, nil
}

func getSecretValue(cl kubernetes.Interface, ns string, env v1alpha1.SecretVar) (string, error) {
	if env.Value != nil {
		return *env.Value, nil
	}

	secret, err := cl.Core().Secrets(ns).Get(env.ValueFrom.Name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	data, ok := secret.Data[env.ValueFrom.Key]
	if !ok {
		return "", fmt.Errorf("Secret %s for `%s` not found", env.ValueFrom.Key, env.ValueFrom.Name)
	}

	return string(data), nil
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
func (c *Client) Update(id string, spec v1alpha1.MonitorTemplateSpec) (string, error) {
	iid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return id, err
	}

	translation, err := c.translateSpec(spec)
	if err != nil {
		return id, err
	}

	translation.TestID = int(iid)
	sct, err := c.cl.Update(translation)

	if err != nil && err.Error() == fmt.Sprintf("No matching key can be found on this account. Given: %s", id) {
		// XXX see if the API returns a 404 HTTP StatusCode, if so, we should add
		// our own client to add proper error handling. This will do for now.
		translation.TestID = 0
		sct, err = c.cl.Update(translation)
	} else if err != nil && err.Error() == fmt.Sprintf("No data has been updated (is any data different?) Given: %s", id) {
		// XXX see if we get a proper status code back for this kind of error,
		// it's not really an error.
		// We want to keep doing these calls to ensure that the monitor is how
		// it should be configured in our specs.
		return id, nil
	}

	if err != nil {
		return id, err
	}

	return strconv.Itoa(sct.TestID), nil
}

// translateSpec does the actual translation from a MonitorTemplateSpec to a
// StatusCake Test.
func (c *Client) translateSpec(spec v1alpha1.MonitorTemplateSpec) (*statuscake.Test, error) {
	scTest := &statuscake.Test{
		WebsiteName:  spec.Name,
		TestType:     spec.Type,
		ContactGroup: c.groups,
		StatusCodes:  statusCodes,
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
