package configure

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/evcc-io/evcc/util/templates"
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
			fmt.Println(c.localizedString("Cancel", nil))
			os.Exit(0)
		}
		fmt.Printf("%s %s\n", c.localizedString("InputError", nil), err)
	}

	return err
}

func (c *CmdConfigure) askConfigFailureNextStep() bool {
	fmt.Println()
	return c.askYesNo(c.localizedString("TestingDevice_RepeatStep", nil))
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
func (c *CmdConfigure) selectItem(deviceCategory DeviceCategory) templates.Template {
	var emptyItem templates.Template
	emptyItem.Description = c.localizedString("ItemNotPresent", nil)

	elements := c.fetchElements(deviceCategory)
	elements = append(elements, emptyItem)

	var items []string
	for _, item := range elements {
		if item.Description != "" {
			items = append(items, item.Description)
		}
	}

	text := fmt.Sprintf("%s %s %s:", c.localizedString("Choose", nil), DeviceCategories[deviceCategory].article, DeviceCategories[deviceCategory].title)
	err, _, selected := c.askSelection(text, items)
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
		value := val.(string)
		if q.invalidValues != nil && funk.ContainsString(q.invalidValues, value) {
			return errors.New(c.localizedString("ValueError_Used", nil))
		}

		if q.required && len(value) == 0 {
			return errors.New(c.localizedString("ValueError_Empty", nil))
		}

		if q.valueType == templates.ParamValueTypeBool {
			if strings.ToLower(value) != "true" && strings.ToLower(value) != "false" {
				return errors.New(c.localizedString("ValueError_Bool", nil))
			}
		}

		if q.valueType == templates.ParamValueTypeFloat {
			_, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return errors.New(c.localizedString("ValueError_Float", nil))
			}
		}

		if q.valueType == templates.ParamValueTypeNumber {
			_, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return errors.New(c.localizedString("ValueError_Number", nil))
			}
		}

		return nil
	}

	help := q.help
	if q.required {
		help += " (" + c.localizedString("Value_Required", nil) + ")"
	} else {
		help += " (" + c.localizedString("Value_Optional", nil) + ")"
	}
	if q.exampleValue != "" {
		help += fmt.Sprintf(" ("+c.localizedString("Value_Sample", nil)+": %s)", q.exampleValue)
	}
	if q.valueType == templates.ParamValueTypeBool {
		help += " (" + c.localizedString("Value_Bool", nil) + ")"
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
			switch q.defaultValue.(type) {
			case string:
				prompt.Default = q.defaultValue.(string)
			case int:
				prompt.Default = strconv.Itoa(q.defaultValue.(int))
			case bool:
				prompt.Default = strconv.FormatBool(q.defaultValue.(bool))
			}
		}
		err = c.surveyAskOne(prompt, &input, survey.WithValidator(validate))
	}

	if err != nil {
		c.log.FATAL.Fatal(err)
	}

	return input
}
