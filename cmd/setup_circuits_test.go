package cmd

import (
	"strings"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/meter"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/tariff"
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

	suite.Require().NoError(viper.ReadConfig(strings.NewReader(`circuits:
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
	suite.Require().Len(lps, 1)
	suite.Require().NotNil(lps[0].GetCircuit())
}

func (suite *circuitsTestSuite) TestLoadpointMissingCircuitError() {
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

	site := core.NewSite()
	circuit.EXPECT().HasMeter().Return(false)
	suite.Require().Error(validateCircuits(site, lps))
}

func (suite *circuitsTestSuite) TestSiteMissingCircuitError() {
	var conf globalconfig.All
	viper.SetConfigType("yaml")

	suite.Require().NoError(viper.ReadConfig(strings.NewReader(`
loadpoints:
- charger: test
site:
  meters:
    grid: grid
`)))

	suite.Require().NoError(viper.UnmarshalExact(&conf))

	lps := []*core.Loadpoint{
		new(core.Loadpoint),
	}

	// mock circuit
	suite.Require().NoError(config.Circuits().Add(config.NewStaticDevice(config.Named{
		Name: "test",
	}, api.Circuit(nil))))

	// mock meter
	m, _ := meter.NewConfigurable(func() (float64, error) {
		return 0, nil
	})
	suite.Require().NoError(config.Meters().Add(config.NewStaticDevice(config.Named{
		Name: "grid",
	}, api.Meter(m))))

	// mock charger
	suite.Require().NoError(config.Chargers().Add(config.NewStaticDevice(config.Named{
		Name: "test",
	}, api.Charger(nil))))

	_, err := configureSite(conf.Site, lps, new(tariff.Tariffs))
	suite.Require().Error(err)
}
