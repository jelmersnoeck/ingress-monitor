package statuscake

import (
	"errors"
	"reflect"
	"strconv"
	"testing"

	"github.com/jelmersnoeck/ingress-monitor/apis/ingressmonitor/v1alpha1"

	"github.com/DreamItGetIT/statuscake"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetSecretVar(t *testing.T) {
	t.Run("with plaintext value", func(t *testing.T) {
		sv := v1alpha1.SecretVar{
			Value: ptrString("plaintext"),
		}

		val, err := getSecretValue(nil, "", sv)
		if err != nil {
			t.Errorf("Expected no error, got %s", err)
		}

		if val != "plaintext" {
			t.Errorf("Expected secret value to be `plaintext`, got `%s`", val)
		}
	})

	t.Run("with reference value", func(t *testing.T) {
		t.Run("with non existing secret", func(t *testing.T) {
			k8s := fake.NewSimpleClientset()
			sv := v1alpha1.SecretVar{
				ValueFrom: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{
						Name: "non-existing",
					},
				},
			}

			_, err := getSecretValue(k8s, "", sv)
			if err == nil {
				t.Errorf("Expected error, got none")
			}
		})

		t.Run("with existing secret", func(t *testing.T) {
			secret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "testing",
				},
				Data: map[string][]byte{
					"username": []byte("my-username"),
				},
			}
			k8s := fake.NewSimpleClientset(secret)

			t.Run("with non existing key", func(t *testing.T) {
				sv := v1alpha1.SecretVar{
					ValueFrom: &v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{
							Name: "test-secret",
						},
						Key: "non-existing",
					},
				}

				_, err := getSecretValue(k8s, "testing", sv)
				if err == nil {
					t.Errorf("Expected error, got none")
				}
			})

			t.Run("in the wrong namespace", func(t *testing.T) {
				sv := v1alpha1.SecretVar{
					ValueFrom: &v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{
							Name: "test-secret",
						},
						Key: "username",
					},
				}

				_, err := getSecretValue(k8s, "wrong-namespace", sv)
				if err == nil {
					t.Errorf("Expected error, got none")
				}
			})

			t.Run("with no errors", func(t *testing.T) {
				sv := v1alpha1.SecretVar{
					ValueFrom: &v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{
							Name: "test-secret",
						},
						Key: "username",
					},
				}

				value, err := getSecretValue(k8s, "testing", sv)
				if err != nil {
					t.Fatalf("Expected no error, got %s", err)
				}

				if value != "my-username" {
					t.Errorf("Expected username to be `my-username`, got `%s`", value)
				}
			})
		})
	})
}

func TestTranslateSpec(t *testing.T) {
	tcs := []struct {
		name     string
		spec     v1alpha1.MonitorTemplateSpec
		groups   []string
		expected *statuscake.Test
	}{
		{
			"simple HTTP config",
			v1alpha1.MonitorTemplateSpec{
				Type: "HTTP",
				HTTP: &v1alpha1.HTTPTemplate{
					CustomHeader: "Test-Header",
					UserAgent:    "(Test User Agent)", URL: "http://fully-qualified-url.com",
					FollowRedirects: true,
				},
			},
			nil,
			&statuscake.Test{
				TestType:       "HTTP",
				CustomHeader:   "Test-Header",
				UserAgent:      "(Test User Agent)",
				WebsiteURL:     "http://fully-qualified-url.com",
				FollowRedirect: true,
				EnableSSLAlert: false,
			},
		},
		{
			"HTTP config should contain",
			v1alpha1.MonitorTemplateSpec{
				Type: "HTTP",
				HTTP: &v1alpha1.HTTPTemplate{
					CustomHeader:      "Test-Header",
					UserAgent:         "(Test User Agent)",
					URL:               "http://fully-qualified-url.com",
					FollowRedirects:   true,
					ShouldContain:     "this string",
					VerifyCertificate: true,
				},
			},
			nil,
			&statuscake.Test{
				TestType:       "HTTP",
				CustomHeader:   "Test-Header",
				UserAgent:      "(Test User Agent)",
				WebsiteURL:     "http://fully-qualified-url.com",
				FollowRedirect: true,
				FindString:     "this string",
				DoNotFind:      false,
				EnableSSLAlert: true,
			},
		},
		{
			"HTTP config should not contain",
			v1alpha1.MonitorTemplateSpec{
				Type: "HTTP",
				HTTP: &v1alpha1.HTTPTemplate{
					CustomHeader:      "Test-Header",
					UserAgent:         "(Test User Agent)",
					URL:               "http://fully-qualified-url.com",
					FollowRedirects:   true,
					ShouldNotContain:  "this string",
					VerifyCertificate: true,
				},
			},
			nil,
			&statuscake.Test{
				TestType:       "HTTP",
				CustomHeader:   "Test-Header",
				UserAgent:      "(Test User Agent)",
				WebsiteURL:     "http://fully-qualified-url.com",
				FollowRedirect: true,
				FindString:     "this string",
				DoNotFind:      true,
				EnableSSLAlert: true,
			},
		},
		{
			"HTTP config with contact groups",
			v1alpha1.MonitorTemplateSpec{
				Type: "HTTP",
				HTTP: &v1alpha1.HTTPTemplate{
					CustomHeader:      "Test-Header",
					UserAgent:         "(Test User Agent)",
					URL:               "http://fully-qualified-url.com",
					FollowRedirects:   true,
					VerifyCertificate: true,
				},
			},
			[]string{"12345"},
			&statuscake.Test{
				TestType:       "HTTP",
				CustomHeader:   "Test-Header",
				UserAgent:      "(Test User Agent)",
				WebsiteURL:     "http://fully-qualified-url.com",
				FollowRedirect: true,
				ContactGroup:   []string{"12345"},
				EnableSSLAlert: true,
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			cl := Client{groups: tc.groups}
			translation, err := cl.translateSpec(tc.spec)
			if err != nil {
				t.Errorf("Expected no error, got %s", err)
				return
			}

			exp := tc.expected
			exp.StatusCodes = statusCodes

			if !reflect.DeepEqual(translation, exp) {
				t.Errorf("Expected translation to equal \n%#v\ngot\n%#v", exp, translation)
			}
		})
	}
}

func TestClient_Delete(t *testing.T) {
	fc := new(fakeClient)
	cl := &Client{cl: fc}

	t.Run("without an error", func(t *testing.T) {
		defer fc.flush()

		fc.deleteFunc = func(i int) error {
			if strconv.Itoa(i) != "12345" {
				t.Errorf("Expected id `12345`, got `%d`", i)
			}

			return nil
		}

		if err := cl.Delete("12345"); err != nil {
			t.Errorf("Expected no error, got %s", err)
		}

		if fc.deleteCount != 1 {
			t.Errorf("Expected 1 delete call, got %d", fc.deleteCount)
		}
	})

	t.Run("with an error", func(t *testing.T) {
		t.Run("invalid number", func(t *testing.T) {
			defer fc.flush()

			if err := cl.Delete("not-a-number"); err == nil {
				t.Errorf("Expected an error, got none")
			}

			if fc.deleteCount != 0 {
				t.Errorf("Expected no delete calls, got %d", fc.deleteCount)
			}
		})

		t.Run("statuscake error", func(t *testing.T) {
			defer fc.flush()

			scError := errors.New("statuscake error")
			fc.deleteFunc = func(i int) error {
				if strconv.Itoa(i) != "12345" {
					t.Errorf("Expected id `12345`, got `%d`", i)
				}

				return scError
			}

			if err := cl.Delete("12345"); err != scError {
				t.Errorf("Expected `%s` error, got `%s`", scError, err)
			}

			if fc.deleteCount != 1 {
				t.Errorf("Expected 1 delete call, got %d", fc.deleteCount)
			}
		})
	})
}

func TestClient_Create(t *testing.T) {
	fc := new(fakeClient)
	cl := &Client{cl: fc}

	t.Run("without error", func(t *testing.T) {
		defer fc.flush()

		tpl := v1alpha1.MonitorTemplateSpec{
			Type: "HTTP",
			HTTP: &v1alpha1.HTTPTemplate{
				CustomHeader: "Test-Header",
				UserAgent:    "(Test User Agent)", URL: "http://fully-qualified-url.com",
				FollowRedirects: true,
			},
		}

		fc.updateFunc = func(sct *statuscake.Test) (*statuscake.Test, error) {
			sct.TestID = 12345
			return sct, nil
		}

		id, err := cl.Create(tpl)
		if err != nil {
			t.Errorf("Expected no error, got %s", err)
		}

		if id != "12345" {
			t.Errorf("Expected ID to be `12345`, got `%s`", id)
		}

		if fc.updateCount != 1 {
			t.Errorf("Expected 1 udpate call, got %d", fc.updateCount)
		}
	})

	t.Run("with translation error", func(t *testing.T) {
		defer fc.flush()

		tpl := v1alpha1.MonitorTemplateSpec{
			Type:    "HTTP",
			Timeout: ptrString("thisisnotvalid"),
			HTTP: &v1alpha1.HTTPTemplate{
				CustomHeader: "Test-Header",
				UserAgent:    "(Test User Agent)", URL: "http://fully-qualified-url.com",
				FollowRedirects: true,
			},
		}

		_, err := cl.Create(tpl)
		if err == nil {
			t.Errorf("Expected error, got none")
		}

		if fc.updateCount != 0 {
			t.Errorf("Expected 0 udpate calls, got %d", fc.updateCount)
		}
	})

	t.Run("with statuscake error", func(t *testing.T) {
		defer fc.flush()

		tpl := v1alpha1.MonitorTemplateSpec{
			Type: "HTTP",
			HTTP: &v1alpha1.HTTPTemplate{
				CustomHeader: "Test-Header",
				UserAgent:    "(Test User Agent)", URL: "http://fully-qualified-url.com",
				FollowRedirects: true,
			},
		}

		scError := errors.New("StatusCakeError")
		fc.updateFunc = func(sct *statuscake.Test) (*statuscake.Test, error) {
			return nil, scError
		}

		_, err := cl.Create(tpl)
		if err != scError {
			t.Errorf("Expected %s error, got %s", scError, err)
		}

		if fc.updateCount != 1 {
			t.Errorf("Expected 1 udpate call, got %d", fc.updateCount)
		}
	})
}

func TestClient_Update(t *testing.T) {
	fc := new(fakeClient)
	cl := &Client{cl: fc}

	t.Run("without error", func(t *testing.T) {
		defer fc.flush()

		tpl := v1alpha1.MonitorTemplateSpec{
			Type: "HTTP",
			HTTP: &v1alpha1.HTTPTemplate{
				CustomHeader: "Test-Header",
				UserAgent:    "(Test User Agent)", URL: "http://fully-qualified-url.com",
				FollowRedirects:   true,
				VerifyCertificate: true,
			},
		}

		fc.updateFunc = func(sct *statuscake.Test) (*statuscake.Test, error) {
			if sct.TestID != 12345 {
				t.Errorf("Expected TestID to be `12345`, got `%d`", sct.TestID)
			}
			return sct, nil
		}

		if _, err := cl.Update("12345", tpl); err != nil {
			t.Errorf("Expected no error, got %s", err)
		}

		if fc.updateCount != 1 {
			t.Errorf("Expected 1 udpate call, got %d", fc.updateCount)
		}
	})

	t.Run("with translation error", func(t *testing.T) {
		defer fc.flush()

		tpl := v1alpha1.MonitorTemplateSpec{
			Type:    "HTTP",
			Timeout: ptrString("thisisnotvalid"),
			HTTP: &v1alpha1.HTTPTemplate{
				CustomHeader: "Test-Header",
				UserAgent:    "(Test User Agent)", URL: "http://fully-qualified-url.com",
				FollowRedirects:   true,
				VerifyCertificate: true,
			},
		}

		if _, err := cl.Update("12345", tpl); err == nil {
			t.Errorf("Expected error, got none")
		}

		if fc.updateCount != 0 {
			t.Errorf("Expected 0 udpate calls, got %d", fc.updateCount)
		}
	})

	t.Run("with statuscake error", func(t *testing.T) {
		defer fc.flush()

		tpl := v1alpha1.MonitorTemplateSpec{
			Type: "HTTP",
			HTTP: &v1alpha1.HTTPTemplate{
				CustomHeader: "Test-Header",
				UserAgent:    "(Test User Agent)", URL: "http://fully-qualified-url.com",
				FollowRedirects:   true,
				VerifyCertificate: true,
			},
		}

		scError := errors.New("StatusCakeError")
		fc.updateFunc = func(sct *statuscake.Test) (*statuscake.Test, error) {
			return nil, scError
		}

		if _, err := cl.Update("12345", tpl); err != scError {
			t.Errorf("Expected %s error, got %s", scError, err)
		}

		if fc.updateCount != 1 {
			t.Errorf("Expected 1 update call, got %d", fc.updateCount)
		}
	})

	t.Run("with changed fields", func(t *testing.T) {
		defer fc.flush()

		tpl := v1alpha1.MonitorTemplateSpec{
			Type: "HTTP",
			HTTP: &v1alpha1.HTTPTemplate{
				CustomHeader: "Test-Header",
				UserAgent:    "(Test User Agent)", URL: "http://fully-qualified-url.com",
				FollowRedirects:   true,
				VerifyCertificate: true,
			},
		}

		fc.updateFunc = func(sct *statuscake.Test) (*statuscake.Test, error) {
			if sct.TestID != 12345 {
				t.Errorf("Expected TestID to be `12345`, got `%d`", sct.TestID)
			}

			// StatusCake sets the ID to 0 if there's changes.
			sct.TestID = 0
			return sct, nil
		}

		id, err := cl.Update("12345", tpl)
		if err != nil {
			t.Errorf("Expected no error, got %s", err)
		}

		if fc.updateCount != 1 {
			t.Errorf("Expected 1 udpate call, got %d", fc.updateCount)
		}

		if id != "12345" {
			t.Errorf("Expected ID to be `12345`, got `%s`", id)
		}
	})
}

type fakeClient struct {
	deleteFunc  func(int) error
	deleteCount int

	updateFunc  func(*statuscake.Test) (*statuscake.Test, error)
	updateCount int
}

func (c *fakeClient) Delete(i int) error {
	c.deleteCount++
	return c.deleteFunc(i)
}

func (c *fakeClient) Update(t *statuscake.Test) (*statuscake.Test, error) {
	c.updateCount++
	return c.updateFunc(t)
}

func (c *fakeClient) flush() {
	c.deleteFunc = nil
	c.deleteCount = 0

	c.updateFunc = nil
	c.updateCount = 0
}

func ptrString(s string) *string {
	return &s
}
