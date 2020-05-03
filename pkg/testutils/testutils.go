package testutils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	jd "github.com/josephburnett/jd/lib"
	"github.com/prometheus/alertmanager/notify/webhook"
)

// ParseWebhookJSONFromFile is a helper for parsing webhook data from JSON files.
func ParseWebhookJSONFromFile(f string) (webhook.Message, error) {
	b, err := ioutil.ReadFile(f)
	if err != nil {
		return webhook.Message{}, err
	}
	var w webhook.Message
	if err := json.Unmarshal(b, &w); err != nil {
		return webhook.Message{}, err
	}
	return w, err
}

// CompareToGoldenFile compares the value of v to file in bytes.
// If update is true, it will update the the golden file using the value of v.
func CompareToGoldenFile(t *testing.T, v interface{}, file string, update bool) {
	gotBytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	gp := filepath.Join("testdata", file)
	dir := filepath.Dir(gp)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		_ = os.MkdirAll(dir, 0755)
	}
	if _, err := os.Stat(gp); os.IsNotExist(err) {
		_ = ioutil.WriteFile(gp, []byte{}, 0644)
	}
	if update {
		t.Log("updating golden file")
		if err := ioutil.WriteFile(gp, gotBytes, 0644); err != nil {
			t.Fatalf("failed to update golden file: %s", err)
		}
	}
	want, err := ioutil.ReadFile(gp)
	if err != nil {
		t.Fatalf("failed reading the golden file: %s", err)
	}
	if string(want) != string(gotBytes) {
		a, err := jd.ReadJsonString(string(gotBytes))
		if err != nil {
			t.Fatal(err)
		}
		b, err := jd.ReadJsonString(string(want))
		if err != nil {
			t.Fatal(err)
		}

		result := fmt.Sprintf(
			"\ngot:\n%s\nwant:\n%s\ndiff:\n%s",
			string(gotBytes),
			string(want),
			a.Diff(b).Render(),
		)

		t.Fatal(result)
	}
}
