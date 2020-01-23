// (C) Copyright 2019 Hewlett Packard Enterprise Development LP.

package update

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const (
	contentType = "Content-Type"
	jsonType    = "application/json"
)

func TestIsUpdateAvailableEmptyLocalVersion(t *testing.T) {
	cases := []struct {
		name       string
		localVer   string
		remoteJSON string
		update     bool
	}{
		{
			name:       "no local version",
			localVer:   "",
			remoteJSON: `{"version":"0.0.0"}`,
			update:     false,
		},
		{
			name:       "no local version",
			localVer:   "",
			remoteJSON: `{"version":"0.0.1"}`,
			update:     true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			server := newTestServer(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(contentType, jsonType)
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, c.remoteJSON)
			})
			defer server.Close()
			got := IsUpdateAvailable()
			if got != c.update {
				t.Fatal("didn't get expected response")
			}
		})
	}
}

func TestIsUpdateAvailablInvalidURLErrors(t *testing.T) {
	versionURL = "://badScheme"
	got := IsUpdateAvailable()
	if got != false {
		t.Fatal("error in checkUpdate should generate false response")
	}
}

func TestCheckSkippedWithEnvSet(t *testing.T) {
	os.Setenv(EnvDisableUpdateCheck, "true")
	defer os.Unsetenv(EnvDisableUpdateCheck)

	got, _ := checkUpdate(&jsonSource{url: ""}, "")
	want := &CheckResponse{}

	//should return empty response because we skip everyting
	//when the env var is set

	validate(t, got, want)
}

func TestCheckUpdate(t *testing.T) {

	cases := []struct {
		name        string
		localVer    string
		remoteJSON  string
		errExpected bool
		want        *CheckResponse
	}{
		{
			name:       "remote greater than local",
			localVer:   "0.0.1",
			remoteJSON: `{"version":"0.1.0"}`,
			want: &CheckResponse{
				UpdateAvailable: true,
				RemoteVersion:   "0.1.0",
			},
		},
		{
			name:       "remote less than local",
			localVer:   "0.0.2",
			remoteJSON: `{"version":"0.0.1"}`,
			want: &CheckResponse{
				UpdateAvailable: false,
				RemoteVersion:   "0.0.1",
			},
		},
		{
			name:       "check all fields",
			localVer:   "0.1.2",
			remoteJSON: `{"version":"0.1.1","message":"update available","url":"https://foo.bar/update","publickey":"00001111","checksum":"120EA8A25E5D487BF68B5F7096440019"}`,
			want: &CheckResponse{
				UpdateAvailable: false,
				RemoteVersion:   "0.1.1",
				Message:         "update available",
				URL:             "https://foo.bar/update",
				PublicKey:       []byte("00001111"),
				CheckSum:        "120EA8A25E5D487BF68B5F7096440019",
			},
		},
		{
			name:        "missing local version",
			localVer:    "",
			errExpected: true,
		},
		{
			name:        "missing remote version",
			localVer:    "0.0.1",
			remoteJSON:  `{"message":"test will fail"}`,
			errExpected: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			server := newTestServer(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(contentType, jsonType)
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, c.remoteJSON)
			})
			defer server.Close()

			json := &jsonSource{
				url: versionURL,
			}

			got, err := checkUpdate(json, c.localVer)
			if err != nil {
				if c.errExpected {
					// got an error.. and expected an error
					return
				}
				// got an error but didn't expect it
				t.Fatal(err)
			}
			validate(t, got, c.want)
		})
	}
}

func validate(t *testing.T, got *CheckResponse, want *CheckResponse) {
	const tmpl = "got: %v, wanted: %v"
	if got.UpdateAvailable != want.UpdateAvailable {
		t.Fatal(fmt.Sprintf(tmpl, got.UpdateAvailable, want.UpdateAvailable))
	}
	if got.RemoteVersion != want.RemoteVersion {
		t.Fatal(fmt.Sprintf(tmpl, got.RemoteVersion, want.RemoteVersion))
	}
	if got.Message != want.Message {
		t.Fatal(fmt.Sprintf(tmpl, got.Message, want.Message))
	}
	if got.URL != want.URL {
		t.Fatal(fmt.Sprintf(tmpl, got.URL, want.URL))
	}
	if bytes.Compare(got.PublicKey, want.PublicKey) != 0 {
		t.Fatal(fmt.Sprintf(tmpl, got.PublicKey, want.PublicKey))
	}
	if got.CheckSum != want.CheckSum {
		t.Fatal(fmt.Sprintf(tmpl, got.CheckSum, want.CheckSum))
	}

}

func newTestServer(h func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	versionURL = fmt.Sprintf("%s%s", server.URL, versionPath)
	mux.HandleFunc(versionPath, h)
	return server
}
