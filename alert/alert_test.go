// Copyright © 2018 bzon <bryansazon@hotmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package alert

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAlertManagerHandler(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(PrometheusAlertManagerHandler))
	defer ts.Close()

	t.Run("only post method is allowed", func(t *testing.T) {
		res, err := http.Get(ts.URL + "/alertmanager")
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Got %d; want %d", res.StatusCode, http.StatusMethodNotAllowed)
		}
	})

	testTable := []struct {
		promAlertJSON  string
		teamsJSON      string
		postStatusCode int
	}{
		{
			promAlertJSON: "examples/prom_post_request.json",
			// Currently not in used..
			teamsJSON: "examples/teams_post_request.json",
			// The Post request will fail because TEAMS_INCOMING_WEBHOOK_URL is not set properly
			postStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range testTable {
		promAlertInBytes, err := ioutil.ReadFile(tc.promAlertJSON)
		if err != nil {
			t.Fatal(err)
		}
		t.Run("decode prometheus alert json", func(t *testing.T) {
			var p PrometheusAlertMessage
			promBuffer := bytes.NewBuffer(promAlertInBytes)
			err = p.DecodePrometheusAlert(promBuffer)
			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("send prometheus alert json", func(t *testing.T) {
			promBuffer := bytes.NewBuffer(promAlertInBytes)
			res, err := http.Post(ts.URL+"/alertmanager", "application/json", promBuffer)
			if err != nil {
				t.Fatal(err)
			}
			if res.StatusCode != tc.postStatusCode {
				t.Fatalf("Want status code %d; got %d", tc.postStatusCode, res.StatusCode)
			}
		})
	}

}
