package card

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/prometheus/alertmanager/notify/webhook"
	"github.com/prometheus/alertmanager/template"
)

// templatedCard implements Converter using Alert manager templating.
type templatedCard struct {
	template *template.Template
}

// NewTemplatedCardCreator creates a templatedCard.
func NewTemplatedCardCreator(t *template.Template) Converter {
	return &templatedCard{t}
}

// Constants for creating a Card
const (
	// Maximum message size of 14336 Bytes (14KB)
	maxSize = 14336
	// Maximum number of sections
	// ref: https://docs.microsoft.com/en-us/microsoftteams/platform/concepts/cards/cards-reference#notes-on-the-office-365-connector-card
	maxCardSections = 10
)

func (m *templatedCard) Convert(promAlert webhook.Message) (JSON, error) {
	data := &template.Data{
		Receiver:          promAlert.Receiver,
		Status:            promAlert.Status,
		Alerts:            promAlert.Alerts,
		GroupLabels:       promAlert.GroupLabels,
		CommonLabels:      promAlert.CommonLabels,
		CommonAnnotations: promAlert.CommonAnnotations,
		ExternalURL:       promAlert.ExternalURL,
	}
	totalMessage, err := m.template.ExecuteTextString(
		`{{ template "teams.card" . }}`, data,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to template alerts: %w", err)
	}

	var (
		cardTmp          string
		restOfMessageTmp string
	)

	cards := "["
	card, restOfMessage, err := splitTooLargeMessage([]byte(totalMessage))
	if err != nil {
		return nil, fmt.Errorf("create card failed: %w", err)
	}
	cards += card

	missingSections, err := querySections(restOfMessage)
	if err != nil {
		return nil, fmt.Errorf("create card failed: %w", err)
	}

	for string(missingSections) != "[]" {
		cardTmp, restOfMessageTmp, err = splitTooLargeMessage([]byte(restOfMessage))
		if err != nil {
			return nil, fmt.Errorf("create card failed: %w", err)
		}
		cards += "," + cardTmp
		restOfMessage = restOfMessageTmp
		missingSections, err = querySections(restOfMessage)
		if err != nil {
			return nil, fmt.Errorf("create card failed: %w", err)
		}
	}
	cards += "]"

	var v JSON
	if err := json.Unmarshal([]byte(cards), &v); err != nil {
		return nil, fmt.Errorf("failed encoding JSON string - '%s' got error: %w", cards, err)
	}
	return v, nil
}

func counter() func() int {
	i := 0
	return func() int {
		i++
		return i
	}
}

func compact(data []byte) (string, error) {
	buffer := new(bytes.Buffer)
	err := json.Compact(buffer, data)
	if err != nil {
		return "", fmt.Errorf("Error calling json.Compact: %w", err)
	}
	return buffer.String(), nil
}

func concatKeyValue(key string, val string) string {
	if strings.HasPrefix(val, "[{") {
		return "\"" + key + "\":" + val
	}
	return "\"" + key + "\":\"" + val + "\""
}

func messageWithoutSections(data []byte) string {
	messageWithoutSections := "{"
	c := counter()
	jsonparser.ObjectEach(
		data,
		func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
			if string(key) != "sections" {
				if c() != 1 {
					messageWithoutSections += ","
				}
				messageWithoutSections += concatKeyValue(string(key), string(value))
			}
			return nil
		},
	)
	messageWithoutSections += "}"
	return messageWithoutSections
}

func splitTooLargeMessage(data []byte) (string, string, error) {
	// finalMessage is a valid Teams message card
	finalMessage := "{"
	// restOfMessage is used to recursively apply this method and iteratively create valid Teams message cards
	restOfMessage := "{"

	length := len(messageWithoutSections(data))

	// range over each key-value pair in the original message card
	c1 := counter()

	objEachErr := jsonparser.ObjectEach(
		data,
		func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
			if string(key) != "sections" {
				if c1() != 1 {
					finalMessage += ","
					restOfMessage += ","
				}
				finalMessage += concatKeyValue(string(key), string(value))
				restOfMessage += concatKeyValue(string(key), string(value))
			}

			if string(key) == "sections" {
				if c1() != 1 {
					finalMessage += ","
					restOfMessage += ","
					length++
				}
				startSections := "\"" + string(key) + "\":["
				finalMessage += startSections
				restOfMessage += startSections
				length++ // for the "]" at the end of the array
				length += len(startSections)
				// counter over section array elements of finalMessage
				c2 := counter()
				// counter over section array elements of restOfMessage
				c3 := counter()
				var counter int
				_, arrayEachErr := jsonparser.ArrayEach(
					value,
					func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
						section, compactErr := compact(value)
						if compactErr != nil {
							err = fmt.Errorf("failed using compact within ArrayEach: %w", err)
							return
						}

						length += len(section)
						counter = c2()
						if counter != 1 {
							length++ // for the leading comma sign before appending a new array element
						}

						if (length <= maxSize) && (counter <= maxCardSections) {
							if counter != 1 {
								finalMessage += ","
							}
							finalMessage += section
						} else {
							if c3() != 1 {
								restOfMessage += ","
							}
							restOfMessage += section
						}
					},
				)
				if arrayEachErr != nil {
					return fmt.Errorf("failed on ArrayEach: %w", arrayEachErr)
				}
				finalMessage += "]"
				restOfMessage += "]"
			}
			return nil
		},
	)
	if objEachErr != nil {
		return "", "", fmt.Errorf("failed on ObjectEach: %w", objEachErr)
	}

	finalMessage += "}"
	restOfMessage += "}"
	return finalMessage, restOfMessage, nil
}

func querySections(message string) ([]byte, error) {
	sections, _, _, err := jsonparser.Get([]byte(message), "sections")
	if err != nil {
		return nil, fmt.Errorf("failed getting query sections: %w", err)
	}
	return sections, nil
}

// ParseTemplateFile creates an alertmanager template from the given file.
func ParseTemplateFile(f string) (*template.Template, error) {
	funcs := template.DefaultFuncs
	funcs["counter"] = func() func() int {
		i := -1
		return func() int {
			i++
			return i
		}
	}
	template.DefaultFuncs = funcs

	if _, err := os.Stat(f); os.IsNotExist(err) {
		return nil, fmt.Errorf("template file %s does not exist", f)
	}

	tmpl, err := template.FromGlobs(f)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse template: %v: %v", err, err)
	}

	return tmpl, nil
}
