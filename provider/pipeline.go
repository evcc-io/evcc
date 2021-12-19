package provider

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/evcc-io/evcc/provider/javascript"
	"github.com/evcc-io/evcc/util"
	"github.com/itchyny/gojq"
	"github.com/robertkrimen/otto"
)

type Pipeline struct {
	re     *regexp.Regexp
	jq     *gojq.Query
	unpack string
	decode string
	vm     *otto.Otto
	script string
}

func NewPipelineFromConfig(other map[string]interface{}) (*Pipeline, error) {
	cc := struct {
		Regex  string
		Jq     string
		Unpack string
		Decode string
		VM     string
		Script string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	p := new(Pipeline)

	var err error
	if err == nil && cc.Regex != "" {
		_, err = p.WithRegex(cc.Regex)
	}

	if err == nil && cc.Jq != "" {
		_, err = p.WithJq(cc.Jq)
	}

	if err == nil && cc.Unpack != "" {
		_, err = p.WithUnpack(cc.Unpack)
	}

	if err == nil && cc.Decode != "" {
		_, err = p.WithDecode(cc.Decode)
	}

	if err == nil && cc.Script != "" {
		_, err = p.WithScript(cc.VM, cc.Script)
	}

	return p, err
}

// WithRegex adds a regex query applied to the mqtt listener payload
func (p *Pipeline) WithRegex(regex string) (*Pipeline, error) {
	re, err := regexp.Compile(regex)
	if err != nil {
		return nil, fmt.Errorf("invalid regex '%s': %w", re, err)
	}

	p.re = re

	return p, nil
}

// WithJq adds a jq query applied to the mqtt listener payload
func (p *Pipeline) WithJq(jq string) (*Pipeline, error) {
	op, err := gojq.Parse(jq)
	if err != nil {
		return nil, fmt.Errorf("invalid jq query '%s': %w", jq, err)
	}

	p.jq = op

	return p, nil
}

// WithUnpack adds data unpacking
func (p *Pipeline) WithUnpack(unpack string) (*Pipeline, error) {
	p.unpack = strings.ToLower(unpack)

	return p, nil
}

// WithDecode adds data decoding
func (p *Pipeline) WithDecode(decode string) (*Pipeline, error) {
	p.decode = strings.ToLower(decode)

	return p, nil
}

// WithScript adds a javascript script to process the response
func (p *Pipeline) WithScript(vm, script string) (*Pipeline, error) {
	regvm := javascript.RegisteredVM(strings.ToLower(vm))

	p.vm = regvm
	p.script = script

	return p, nil
}
