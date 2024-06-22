package cmd

import (
	"strings"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/globalconfig"
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

	suite.Require().NoError(configureCircuits(conf.Circuits))
	suite.Require().Len(config.Circuits().Devices(), 2)
	suite.Require().False(config.Circuits().Devices()[0].Instance().HasMeter())

	// empty charger
	suite.Require().NoError(config.Chargers().Add(config.NewStaticDevice(config.Named{
		Name: "test",
	}, api.Charger(nil))))

	lps, err := configureLoadpoints(conf)
	suite.Require().NoError(err)
	suite.Require().NotNil(lps[0].GetCircuit())
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

	suite.Require().NoError(configureCircuits(conf.Circuits))
	suite.Require().Len(config.Circuits().Devices(), 2)
	suite.Require().False(config.Circuits().Devices()[0].Instance().HasMeter())

	// empty charger
	suite.Require().NoError(config.Chargers().Add(config.NewStaticDevice(config.Named{
		Name: "test",
	}, api.Charger(nil))))

	lps, err := configureLoadpoints(conf)
	suite.Require().NoError(err)

	// circuit without device
	err = validateCircuits(lps)
	suite.Require().Error(err)
	suite.Require().Equal("circuit slave has no meter and no loadpoint assigned", err.Error())
}

func (suite *circuitsTestSuite) TestMissingRootCircuit() {
	var conf globalconfig.All
	viper.SetConfigType("yaml")

	suite.Require().NoError(viper.ReadConfig(strings.NewReader(`
loadpoints:
- charger: test
`)))

	suite.Require().NoError(viper.UnmarshalExact(&conf))

	ctrl := gomock.NewController(suite.T())
	circuit := api.NewMockCircuit(ctrl)

	// mock circuit
	suite.Require().NoError(config.Circuits().Add(config.NewStaticDevice(config.Named{
		Name: "test",
	}, api.Circuit(circuit))))

	// mock charger
	suite.Require().NoError(config.Chargers().Add(config.NewStaticDevice(config.Named{
		Name: "test",
	}, api.Charger(nil))))

	lps, err := configureLoadpoints(conf)
	suite.Require().NoError(err)

	// root circuit
	circuit.EXPECT().GetParent().Return(nil)
	circuit.EXPECT().HasMeter().Return(true)
	suite.Require().NoError(validateCircuits(lps))

	// no root circuit
	circuit.EXPECT().GetParent().Return(circuit)
	err = validateCircuits(lps)
	suite.Require().Error(err)
	suite.Require().Equal("missing root circuit", err.Error())
}

func (suite *circuitsTestSuite) TestLoadpointUsingRootCircuit() {
	var conf globalconfig.All
	viper.SetConfigType("yaml")

	suite.Require().NoError(viper.ReadConfig(strings.NewReader(`
loadpoints:
- charger: test
  circuit: root
`)))

	suite.Require().NoError(viper.UnmarshalExact(&conf))

	ctrl := gomock.NewController(suite.T())
	circuit := api.NewMockCircuit(ctrl)

	// mock circuit
	suite.Require().NoError(config.Circuits().Add(config.NewStaticDevice(config.Named{
		Name: "root",
	}, api.Circuit(circuit))))

	// mock charger
	suite.Require().NoError(config.Chargers().Add(config.NewStaticDevice(config.Named{
		Name: "test",
	}, api.Charger(nil))))

	lps, err := configureLoadpoints(conf)
	suite.Require().NoError(err)

	// root circuit
	circuit.EXPECT().GetParent().Return(nil)
	err = validateCircuits(lps)
	suite.Require().Error(err)
	suite.Require().Equal("root circuit must not be assigned to loadpoint ", err.Error())
}
