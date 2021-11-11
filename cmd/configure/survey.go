package configure

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/evcc-io/evcc/templates"
	"github.com/thoas/go-funk"
)

// Survey: ask the user for input
func (c *CmdConfigure) surveyAskOne(p survey.Prompt, response interface{}, opts ...survey.AskOpt) error {
	err := survey.AskOne(p, response, survey.WithIcons(func(icons *survey.IconSet) {
		icons.Question.Text = ""
	}))

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

// Survey: ask for input
func (c *CmdConfigure) askValue(label, exampleValue, hint string, invalidValues []string, dataType string, mask, required bool) string {
	input := ""

	var err error

	validate := func(val interface{}) error {
		if invalidValues != nil && funk.ContainsString(invalidValues, input) {
			return errors.New("Der Wert '" + input + "' wurde bereits verwendet.")
		}

		if required && len(input) == 0 {
			return errors.New("Der Wert darf nicht leer sein")
		}

		if dataType == templates.ParamValueTypeInt {
			_, err := strconv.Atoi(input)
			if err != nil {
				return errors.New("Der Wert muss eine Zahl sein.")
			}
		}

		return nil
	}

	if mask {
		prompt := &survey.Password{
			Message: label,
			Help:    hint,
		}
		err = c.surveyAskOne(prompt, &input, survey.WithValidator(validate))

	} else {
		prompt := &survey.Input{
			Message: label,
			Help:    hint,
		}
		err = c.surveyAskOne(prompt, &input)

	}

	if err != nil {
		c.log.FATAL.Fatal(err)
	}

	return input
}
