package cmd

import (
	"strings"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestSetupCircuits(t *testing.T) {
	suite.Run(t, new(circuitsTestSuite))
}

type circuitsTestSuite struct {
	suite.Suite
}

func (suite *circuitsTestSuite) SetupSuite() {
	db, err := db.New("sqlite", ":memory:")
	if err != nil {
		suite.T().Fatal(err)
	}
	config.Init(db)
}

func (suite *circuitsTestSuite) SetupTest() {
	config.Reset()
}

func (suite *circuitsTestSuite) TestCircuitConf() {
	var conf globalconfig.All
	viper.SetConfigType("yaml")

	suite.Require().NoError(viper.ReadConfig(strings.NewReader(`
circuits:
- name: master
  maxPower: 10000
- name: slave
  parent: master
  maxPower: 10000
loadpoints:
- charger: test
  circuit: slave
`)))

	suite.Require().NoError(viper.UnmarshalExact(&conf))

	suite.Require().NoError(configureCircuits(&conf.Circuits))
	suite.Require().Len(config.Circuits().Devices(), 2)
	suite.Require().False(config.Circuits().Devices()[0].Instance().HasMeter())

	// empty charger
	suite.Require().NoError(config.Chargers().Add(config.NewStaticDevice(config.Named{
		Name: "test",
	}, api.Charger(nil))))

	err := configureLoadpoints(conf)
	suite.Require().NoError(err)

	lps := config.Loadpoints().Devices()
	suite.Require().NotNil(lps[0].Instance().GetCircuit())
}

func (suite *circuitsTestSuite) TestCircuitMissingLoadpoint() {
	var conf globalconfig.All
	viper.SetConfigType("yaml")

	suite.Require().NoError(viper.ReadConfig(strings.NewReader(`
circuits:
- name: master
- name: slave
  parent: master
loadpoints:
- charger: test
`)))

	suite.Require().NoError(viper.UnmarshalExact(&conf))

	suite.Require().NoError(configureCircuits(&conf.Circuits))
	suite.Require().Len(config.Circuits().Devices(), 2)
	suite.Require().False(config.Circuits().Devices()[0].Instance().HasMeter())

	// empty charger
	suite.Require().NoError(config.Chargers().Add(config.NewStaticDevice(config.Named{
		Name: "test",
	}, api.Charger(nil))))

	err := configureLoadpoints(conf)
	suite.Require().NoError(err)

	lpsd := config.Loadpoints().Devices()
	lps := make([]*core.Loadpoint, 0, len(lpsd))
	for _, lp := range lpsd {
		lps = append(lps, lp.Instance().(*core.Loadpoint))
	}

	// circuit without device
	err = validateCircuits(lps)
	suite.Require().NoError(err)
}

func (suite *circuitsTestSuite) TestMissingRootCircuit() {
	ctrl := gomock.NewController(suite.T())
	circuit := api.NewMockCircuit(ctrl)

	// circuit device
	suite.Require().NoError(config.Circuits().Add(config.NewStaticDevice(config.Named{
		Name: "master",
	}, api.Circuit(circuit))))

	// root circuit present
	circuit.EXPECT().GetParent().Return(nil)
	circuit.EXPECT().HasMeter().Return(true)
	suite.Require().NoError(validateCircuits(nil))

	// root circuit missing
	circuit.EXPECT().GetParent().Return(circuit)
	err := validateCircuits(nil)
	suite.Require().Error(err)
	suite.Require().Equal("missing root circuit", err.Error())
}

func (suite *circuitsTestSuite) TestLoadpointUsingRootCircuit() {
	var conf globalconfig.All
	viper.SetConfigType("yaml")

	suite.Require().NoError(viper.ReadConfig(strings.NewReader(`
circuits:
- name: master
loadpoints:
- charger: test
  circuit: master
`)))

	suite.Require().NoError(viper.UnmarshalExact(&conf))

	suite.Require().NoError(configureCircuits(&conf.Circuits))
	suite.Require().Len(config.Circuits().Devices(), 1)

	// mock charger
	suite.Require().NoError(config.Chargers().Add(config.NewStaticDevice(config.Named{
		Name: "test",
	}, api.Charger(nil))))

	err := configureLoadpoints(conf)
	suite.Require().NoError(err)

	lpsd := config.Loadpoints().Devices()
	lps := make([]*core.Loadpoint, 0, len(lpsd))
	for _, lp := range lpsd {
		lps = append(lps, lp.Instance().(*core.Loadpoint))
	}

	// lp using root circuit is valid
	suite.Require().NoError(validateCircuits(lps))
}
