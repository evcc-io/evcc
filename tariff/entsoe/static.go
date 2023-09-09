package entsoe

import (
	"encoding/xml"

	"github.com/evcc-io/evcc/util/shortrfc3339"
)

// This file contains static declarations of structs and constants.
// Heavily derived from https://github.com/energy-forecast/go-entsoe (MIT license), many thanks!

type ProcessType string

const (
	ProcessTypeDayAhead ProcessType = "A44"
)

type DomainType = string

const (
	DomainAL           DomainType = "10YAL-KESH-----5"
	DomainAT           DomainType = "10YAT-APG------L"
	DomainBA           DomainType = "10YBA-JPCC-----D"
	DomainBE           DomainType = "10YBE----------2"
	DomainBG           DomainType = "10YCA-BULGARIA-R"
	DomainBY           DomainType = "10Y1001A1001A51S"
	DomainCH           DomainType = "10YCH-SWISSGRIDZ"
	DomainCZ           DomainType = "10YCZ-CEPS-----N"
	DomainDE           DomainType = "10Y1001A1001A83F"
	DomainDE50Hertz    DomainType = "10YDE-VE-------2"
	DomainDEAmprion    DomainType = "10YDE-RWENET---I"
	DomainDETenneT     DomainType = "10YDE-EON------1"
	DomainDETransnetBW DomainType = "10YDE-ENBW-----N"
	DomainDK           DomainType = "10Y1001A1001A65H"
	DomainEE           DomainType = "10Y1001A1001A39I"
	DomainES           DomainType = "10YES-REE------0"
	DomainFI           DomainType = "10YFI-1--------U"
	DomainFR           DomainType = "10YFR-RTE------C"
	DomainGB           DomainType = "10YGB----------A"
	DomainGBNIR        DomainType = "10Y1001A1001A016"
	DomainGR           DomainType = "10YGR-HTSO-----Y"
	DomainHR           DomainType = "10YHR-HEP------M"
	DomainHU           DomainType = "10YHU-MAVIR----U"
	DomainIE           DomainType = "10YIE-1001A00010"
	DomainIT           DomainType = "10YIT-GRTN-----B"
	DomainLT           DomainType = "10YLT-1001A0008Q"
	DomainLU           DomainType = "10YLU-CEGEDEL-NQ"
	DomainLV           DomainType = "10YLV-1001A00074"
	DomainME           DomainType = "10YCS-CG-TSO---S"
	DomainMK           DomainType = "10YMK-MEPSO----8"
	DomainMT           DomainType = "10Y1001A1001A93C"
	DomainNL           DomainType = "10YNL----------L"
	DomainNO           DomainType = "10YNO-0--------C"
	DomainPL           DomainType = "10YPL-AREA-----S"
	DomainPT           DomainType = "10YPT-REN------W"
	DomainRO           DomainType = "10YRO-TEL------P"
	DomainRS           DomainType = "10YCS-SERBIATSOV"
	DomainRU           DomainType = "10Y1001A1001A49F"
	DomainRUKGD        DomainType = "10Y1001A1001A50U"
	DomainSE           DomainType = "10YSE-1--------K"
	DomainSI           DomainType = "10YSI-ELES-----O"
	DomainSK           DomainType = "10YSK-SEPS-----K"
	DomainTR           DomainType = "10YTR-TEIAS----W"
	DomainUA           DomainType = "10YUA-WEPS-----0"
	DomainDEATLU       DomainType = "10Y1001A1001A63L"
)

type ResolutionType string

const (
	ResolutionQuarterHour ResolutionType = "PT15M"
	ResolutionHalfHour    ResolutionType = "PT30M"
	ResolutionHour        ResolutionType = "PT60M"
	ResolutionDay         ResolutionType = "P1D"
	ResolutionWeek        ResolutionType = "P7D"
	ResolutionYear        ResolutionType = "P1Y"
)

type AttributeInstanceComponent struct {
	XMLName        xml.Name `xml:"AttributeInstanceComponent"`
	Attribute      string   `xml:"attribute"`
	AttributeValue string   `xml:"attributeValue"`
}

type PublicationMarketDocument struct {
	XMLName                     xml.Name `xml:"Publication_MarketDocument"`
	Text                        string   `xml:",chardata"`
	Xmlns                       string   `xml:"xmlns,attr"`
	MRID                        string   `xml:"mRID"`           // abbbeef260884cb9b43858124...
	RevisionNumber              string   `xml:"revisionNumber"` // 1, 1, 1, 1, 1, 1, 1, 1, 1...
	Type                        string   `xml:"type"`           // A44, A25, A25, A09, A11, ...
	SenderMarketParticipantMRID struct {
		Text         string `xml:",chardata"` // 10X1001A1001A450, 10X1001...
		CodingScheme string `xml:"codingScheme,attr"`
	} `xml:"sender_MarketParticipant.mRID"`
	SenderMarketParticipantMarketRoleType string `xml:"sender_MarketParticipant.marketRole.type"` // A32, A32, A32, A32, A32, ...
	ReceiverMarketParticipantMRID         struct {
		Text         string `xml:",chardata"` // 10X1001A1001A450, 10X1001...
		CodingScheme string `xml:"codingScheme,attr"`
	} `xml:"receiver_MarketParticipant.mRID"`
	ReceiverMarketParticipantMarketRoleType string `xml:"receiver_MarketParticipant.marketRole.type"` // A33, A33, A33, A33, A33, ...
	CreatedDateTime                         string `xml:"createdDateTime"`                            // 2020-09-12T00:13:15Z, 202...
	PeriodTimeInterval                      struct {
		Text  string                 `xml:",chardata"`
		Start shortrfc3339.Timestamp `xml:"start"` // 2015-12-31T23:00Z, 2015-1...
		End   shortrfc3339.Timestamp `xml:"end"`   // 2016-12-31T23:00Z, 2016-1...
	} `xml:"period.timeInterval"`
	TimeSeries []TimeSeries `xml:"TimeSeries"`
}
type TimeSeries struct {
	Text         string `xml:",chardata"`
	MRID         string `xml:"mRID"`         // 1, 2, 3, 4, 5, 6, 7, 8, 9...
	BusinessType string `xml:"businessType"` // A62, A62, A62, A62, A62, ...
	InDomainMRID struct {
		Text         string `xml:",chardata"` // 10YCZ-CEPS-----N, 10YCZ-C...
		CodingScheme string `xml:"codingScheme,attr"`
	} `xml:"in_Domain.mRID"`
	OutDomainMRID struct {
		Text         string `xml:",chardata"` // 10YCZ-CEPS-----N, 10YCZ-C...
		CodingScheme string `xml:"codingScheme,attr"`
	} `xml:"out_Domain.mRID"`
	CurrencyUnitName                                         string           `xml:"currency_Unit.name"`      // EUR, EUR, EUR, EUR, EUR, ...
	PriceMeasureUnitName                                     string           `xml:"price_Measure_Unit.name"` // MWH, MWH, MWH, MWH, MWH, ...
	CurveType                                                string           `xml:"curveType"`               // A01, A01, A01, A01, A01, ...
	Period                                                   TimeSeriesPeriod `xml:"Period"`
	AuctionType                                              string           `xml:"auction.type"`                                               // A01, A01, A01, A01, A01, ...
	ContractMarketAgreementType                              string           `xml:"contract_MarketAgreement.type"`                              // A01, A01, A01, A01, A01, ...
	QuantityMeasureUnitName                                  string           `xml:"quantity_Measure_Unit.name"`                                 // MAW, MAW, MAW, MAW, MAW, ...
	AuctionMRID                                              string           `xml:"auction.mRID"`                                               // CP_A_Hourly_SK-UA, CP_A_D...
	AuctionCategory                                          string           `xml:"auction.category"`                                           // A04, A04, A01, A01, A01, ...
	ClassificationSequenceAttributeInstanceComponentPosition string           `xml:"classificationSequence_AttributeInstanceComponent.position"` // 1, 1
}

type TimeSeriesPeriod struct {
	Text         string `xml:",chardata"`
	TimeInterval struct {
		Text  string                 `xml:",chardata"`
		Start shortrfc3339.Timestamp `xml:"start"` // 2015-12-31T23:00Z, 2016-0...
		End   shortrfc3339.Timestamp `xml:"end"`   // 2016-01-01T23:00Z, 2016-0...
	} `xml:"timeInterval"`
	Resolution ResolutionType `xml:"resolution"` // PT60M, PT60M, PT60M, PT60...
	Point      []struct {
		Text        string `xml:",chardata"`
		Position    int    `xml:"position"`     // 1, 2, 3, 4, 5, 6, 7, 8, 9...
		PriceAmount string `xml:"price.amount"` // 16.50, 15.50, 14.00, 10.0...
		Quantity    string `xml:"quantity"`     // 226, 87, 104, 189, 217, 8...
	} `xml:"Point"`
}
