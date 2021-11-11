package configure

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/evcc-io/evcc/templates"
	"github.com/manifoldco/promptui"
	"github.com/thoas/go-funk"
)

// PromptUI: select item from list
func (c *CmdConfigure) selectItem(deviceCategory string) templates.Template {
	promptuiTemplates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "-> {{ .Description }} {{ if .Requirements.Sponsorship }}(Sponsorship benötigt){{ end }}",
		Inactive: "   {{ .Description }} {{ if .Requirements.Sponsorship }}(Sponsorship benötigt){{ end }}",
		Selected: fmt.Sprintf("%s: {{ .Description }}", DeviceCategories[deviceCategory].class),
	}

	var emptyItem templates.Template
	emptyItem.Description = itemNotPresent

	items := c.fetchElements(deviceCategory)
	items = append(items, emptyItem)

	prompt := promptui.Select{
		Label:     fmt.Sprintf("Wähle %s %s", DeviceCategories[deviceCategory].article, DeviceCategories[deviceCategory].title),
		Items:     items,
		Templates: promptuiTemplates,
		Size:      10,
	}

	index, _, err := prompt.Run()
	if err != nil {
		c.log.FATAL.Fatal(err)
	}

	return items[index]
}

// PromptUI: select item from list
func (c *CmdConfigure) askChoice(label string, choices []string) (int, string) {
	promptuiTemplates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "-> {{ . }}",
		Inactive: "   {{ . }}",
		Selected: "   {{ . }}",
	}

	prompt := promptui.Select{
		Label:     label,
		Items:     choices,
		Templates: promptuiTemplates,
		Size:      10,
	}

	index, result, err := prompt.Run()
	if err != nil {
		c.log.FATAL.Fatal(err)
	}

	return index, result
}

// PromptUI: ask yes/no question, return true if yes is selected
func (c *CmdConfigure) askYesNo(label string) bool {
	prompt := promptui.Prompt{
		Label:     label,
		IsConfirm: true,
	}

	_, err := prompt.Run()

	return !errors.Is(err, promptui.ErrAbort)
}

// PromputUI: ask for input
func (c *CmdConfigure) askValue(label, exampleValue, hint string, invalidValues []string, dataType string, mask, required bool) string {
	promptuiTemplates := &promptui.PromptTemplates{
		Prompt:  "{{ . }} ",
		Valid:   "{{ . | green }} ",
		Invalid: "{{ . | red }} ",
		Success: "{{ . | bold }} ",
	}

	validate := func(input string) error {
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

	if hint != "" {
		fmt.Println(hint)
	}

	prompt := promptui.Prompt{
		Label:     label,
		Templates: promptuiTemplates,
		Default:   exampleValue,
		Validate:  validate,
		AllowEdit: true,
	}

	if mask {
		prompt.Mask = '*'
	}

	result, err := prompt.Run()
	if err != nil {
		c.log.FATAL.Fatal(err)
	}

	return result
}
