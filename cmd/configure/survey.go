package configure

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/templates"
	stripmd "github.com/writeas/go-strip-markdown/v2"
)

// surveyAskOne asks the user for input
func (c *CmdConfigure) surveyAskOne(p survey.Prompt, response interface{}, opts ...survey.AskOpt) error {
	opts = append(opts, survey.WithIcons(func(icons *survey.IconSet) {
		icons.Question.Text = ""
	}))
	err := survey.AskOne(p, response, opts...)
	if err != nil {
		if err == terminal.InterruptErr {
			fmt.Println(c.localizedString("Cancel"))
			os.Exit(0)
		}
		fmt.Printf("%s %s\n", c.localizedString("InputError"), err)
	}

	return err
}

// askConfigFailureNextStep asks the user if he/she wants to select another device because the current does not work, or continue
func (c *CmdConfigure) askConfigFailureNextStep() bool {
	fmt.Println()
	return c.askYesNo(c.localizedString("TestingDevice_RepeatStep"))
}

// select item from list
func (c *CmdConfigure) askSelection(message string, items []string) (string, int, error) {
	prompt := &survey.Select{
		Message: message,
		Options: items,
	}

	var selection string
	err := c.surveyAskOne(prompt, &selection)

	return selection, slices.Index(items, selection), err
}

// selectItem selects item from list
func (c *CmdConfigure) selectItem(deviceCategory DeviceCategory) templates.Template {
	var emptyItem templates.Template
	emptyItem.SetTitle(c.localizedString("ItemNotPresent"))

	elements := c.fetchElements(deviceCategory)
	elements = append(elements, emptyItem)

	var items []string
	for _, item := range elements {
		if item.Title() != "" {
			items = append(items, item.Title())
		}
	}

	text := fmt.Sprintf("%s %s %s:", c.localizedString("Choose"), DeviceCategories[deviceCategory].article, DeviceCategories[deviceCategory].title)
	_, selected, err := c.askSelection(text, items)
	if err != nil {
		c.log.FATAL.Fatal(err)
	}

	return elements[selected]
}

// askChoice selects item from list
func (c *CmdConfigure) askChoice(label string, choices []string) (int, string) {
	selection, index, err := c.askSelection(label, choices)
	if err != nil {
		c.log.FATAL.Fatal(err)
	}

	return index, selection
}

// askYesNo asks yes/no question, return true if yes is selected
func (c *CmdConfigure) askYesNo(label string) bool {
	prompt := &survey.Confirm{
		Message: label,
	}

	var confirmation bool
	if err := c.surveyAskOne(prompt, &confirmation); err != nil {
		c.log.FATAL.Fatal(err)
	}

	return confirmation
}

type question struct {
	label, help                    string
	defaultValue, exampleValue     string
	invalidValues                  []string
	validValues                    []string
	valueType                      templates.ParamType
	minNumberValue, maxNumberValue int64
	mask, required                 bool
	excludeNone                    bool
}

// askBoolValue asks for a boolean value selection for a given question
func (c *CmdConfigure) askBoolValue(label string) string {
	choices := []string{c.localizedString("Config_No"), c.localizedString("Config_Yes")}
	values := []string{"false", "true"}

	index, _ := c.askChoice(label, choices)
	return values[index]
}

func (c *CmdConfigure) askParam(p templates.Param) string {
	var mask, required bool
	if p.Mask != nil {
		mask = *p.Mask
	}
	if p.Required != nil {
		required = *p.Required
	}

	return c.askValue(question{
		label:       p.Description.String(c.lang),
		valueType:   p.Type,
		validValues: p.Choice, // TODO proper choice handling
		mask:        mask,
		required:    required,
	})
}

// askValue asks for value input for a given question (template param)
func (c *CmdConfigure) askValue(q question) string {
	if q.valueType == templates.TypeBool {
		label := q.label
		if q.help != "" {
			helpDescription := stripmd.Strip(q.help)
			fmt.Println("-------------------------------------------------")
			fmt.Println(c.localizedString("Value_Help"))
			fmt.Println(helpDescription)
			fmt.Println("-------------------------------------------------")
		}

		return c.askBoolValue(label)
	}

	if q.valueType == templates.TypeChoice {
		label := strings.TrimSpace(strings.Join([]string{q.label, c.localizedString("Value_Choice")}, " "))
		idx, _ := c.askChoice(label, q.validValues)
		return q.validValues[idx]
	}

	if q.valueType == templates.TypeChargeModes {
		chargingModes := []string{string(api.ModeOff), string(api.ModeNow), string(api.ModeMinPV), string(api.ModePV)}
		chargeModes := []string{
			c.localizedString("ChargeModeOff"),
			c.localizedString("ChargeModeNow"),
			c.localizedString("ChargeModeMinPV"),
			c.localizedString("ChargeModePV"),
		}
		if !q.excludeNone {
			chargingModes = append(chargingModes, "")
			chargeModes = append(chargeModes, c.localizedString("ChargeModeNone"))
		}
		modeChoice, _ := c.askChoice(c.localizedString("ChargeMode_Question"), chargeModes)
		return chargingModes[modeChoice]
	}

	validate := func(val interface{}) error {
		value := val.(string)
		if q.invalidValues != nil && slices.Contains(q.invalidValues, value) {
			return errors.New(c.localizedString("ValueError_Used"))
		}

		if q.validValues != nil && !slices.Contains(q.validValues, value) {
			return errors.New(c.localizedString("ValueError_Invalid"))
		}

		if q.required && len(value) == 0 {
			return errors.New(c.localizedString("ValueError_Empty"))
		}

		if !q.required && len(value) == 0 {
			return nil
		}

		if q.valueType == templates.TypeFloat {
			_, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return errors.New(c.localizedString("ValueError_Float"))
			}
		}

		if q.valueType == templates.TypeNumber {
			intValue, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return errors.New(c.localizedString("ValueError_Number"))
			}
			if q.minNumberValue != 0 && intValue < q.minNumberValue {
				return errors.New(c.localizedString("ValueError_NumberLowerThanMin", localizeMap{"Min": q.minNumberValue}))
			}
			if q.maxNumberValue != 0 && intValue > q.maxNumberValue {
				return errors.New(c.localizedString("ValueError_NumberBiggerThanMax", localizeMap{"Max": q.maxNumberValue}))
			}
		}

		if q.valueType == templates.TypeDuration {
			_, err := time.ParseDuration(value)
			if err != nil {
				return errors.New(c.localizedString("ValueError_Duration"))
			}
		}

		return nil
	}

	help := q.help
	if q.required {
		help += " (" + c.localizedString("Value_Required") + ")"
	} else {
		help += " (" + c.localizedString("Value_Optional") + ")"
	}
	if q.exampleValue != "" {
		help += fmt.Sprintf(" ("+c.localizedString("Value_Sample")+": %s)", q.exampleValue)
	}

	var prompt survey.Prompt
	if q.mask {
		prompt = &survey.Password{
			Message: q.label,
			Help:    help,
		}
	} else {
		prompt = &survey.Input{
			Message: q.label,
			Default: q.defaultValue,
			Help:    help,
		}
	}

	var input string
	if err := c.surveyAskOne(prompt, &input, survey.WithValidator(validate)); err != nil {
		c.log.FATAL.Fatal(err)
	}

	return input
}
