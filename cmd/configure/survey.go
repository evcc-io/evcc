package configure

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/evcc-io/evcc/templates"
	"github.com/thoas/go-funk"
)

// Survey: ask the user for input
func (c *CmdConfigure) surveyAskOne(p survey.Prompt, response interface{}, opts ...survey.AskOpt) error {
	opts = append(opts, survey.WithIcons(func(icons *survey.IconSet) {
		icons.Question.Text = ""
	}))
	err := survey.AskOne(p, response, opts...)

	if err != nil {
		if err == terminal.InterruptErr {
			fmt.Println("Konfiguration wurde abgebrochen.")
			fmt.Println()
			fmt.Println("Falls diese Konfiguration für dich noch nicht funktioniert, versuche es doch mal mit der manuellen Konfiguration. Details findest du auf der folgenden Webseite: https://docs.evcc.io/docs/installation/configuration")
			fmt.Println()
			os.Exit(0)
		}
		fmt.Println("Es ist bei der Eingabe ein Fehler aufgetreten: ", err)
	}

	return err
}

func (c *CmdConfigure) askConfigFailureNextStep() bool {
	fmt.Println()
	return c.askYesNo("Die Konfiguration funktioniert leider nicht und kann daher nicht verwendet werden. Möchtest du es nochmals versuchen?")
}

// Survey: select item from list
func (c *CmdConfigure) askSelection(message string, items []string) (error, string, int) {
	selection := ""
	prompt := &survey.Select{
		Message: message,
		Options: items,
	}

	err := c.surveyAskOne(prompt, &selection)
	if err != nil {
		return err, "", 0
	}

	var selectedIndex int
	for index, item := range items {
		if item == selection {
			selectedIndex = index
			break
		}
	}

	return err, selection, selectedIndex
}

// Survey: select item from list
func (c *CmdConfigure) selectItem(deviceCategory string) templates.Template {
	var emptyItem templates.Template
	emptyItem.Description = itemNotPresent

	elements := c.fetchElements(deviceCategory)
	elements = append(elements, emptyItem)

	var items []string
	for _, item := range elements {
		if item.Description != "" {
			items = append(items, item.Description)
		}
	}

	err, _, selected := c.askSelection(fmt.Sprintf("Wähle %s %s:", DeviceCategories[deviceCategory].article, DeviceCategories[deviceCategory].title), items)
	if err != nil {
		c.log.FATAL.Fatal(err)
	}

	return elements[selected]
}

// Survey: select item from list
func (c *CmdConfigure) askChoice(label string, choices []string) (int, string) {
	err, selection, index := c.askSelection(label, choices)
	if err != nil {
		c.log.FATAL.Fatal(err)
	}

	return index, selection
}

// Survey: ask yes/no question, return true if yes is selected
func (c *CmdConfigure) askYesNo(label string) bool {
	confirmation := false
	prompt := &survey.Confirm{
		Message: label,
	}

	err := c.surveyAskOne(prompt, &confirmation)
	if err != nil {
		c.log.FATAL.Fatal(err)
	}

	return confirmation
}

type question struct {
	label, help                string
	defaultValue, exampleValue interface{}
	invalidValues              []string
	valueType                  string
	mask, required             bool
}

// Survey: ask for input
func (c *CmdConfigure) askValue(q question) string {
	input := ""

	var err error

	validate := func(val interface{}) error {
		switch val.(type) {
		case string:
			value := val.(string)
			if q.invalidValues != nil && funk.ContainsString(q.invalidValues, value) {
				return errors.New("Der Wert '" + value + "' wurde bereits verwendet.")
			}

			if q.required && len(value) == 0 {
				return errors.New("Der Wert darf nicht leer sein")
			}

			if q.valueType == templates.ParamValueTypeBool {
				if strings.ToLower(value) != "true" && strings.ToLower(value) != "false" {
					return errors.New("Der Wert muss 'true' (für ja) oder 'false' (für nein) sein.")
				}
			}

			if q.valueType == templates.ParamValueTypeInt {
				_, err := strconv.Atoi(value)
				if err != nil {
					return errors.New("Der Wert muss eine Zahl sein.")
				}
			}
		}

		return nil
	}

	help := q.help
	if q.required {
		help += " (erforderlich)"
	} else {
		help += " (optional)"
	}
	if q.exampleValue != "" {
		help += fmt.Sprintf(" (Beispiel: %s)", q.exampleValue)
	}
	if q.valueType == templates.ParamValueTypeBool {
		help += " ('true' für ja oder 'false' für nein)"
	}

	if q.mask {
		prompt := &survey.Password{
			Message: q.label,
			Help:    help,
		}
		err = c.surveyAskOne(prompt, &input, survey.WithValidator(validate))

	} else {
		prompt := &survey.Input{
			Message: q.label,
			Help:    help,
		}
		if q.defaultValue != nil {
			if q.valueType == templates.ParamValueTypeInt && reflect.TypeOf(q.defaultValue).Kind() == reflect.Int {
				prompt.Default = strconv.Itoa(q.defaultValue.(int))
			} else {
				prompt.Default = q.defaultValue.(string)
			}
		}
		err = c.surveyAskOne(prompt, &input, survey.WithValidator(validate))
	}

	if err != nil {
		c.log.FATAL.Fatal(err)
	}

	return input
}
