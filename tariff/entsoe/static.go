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

const (
	AcknowledgementMarketDocumentName = "Acknowledgement_MarketDocument"
	PublicationMarketDocumentName     = "Publication_MarketDocument"
)

type Document struct {
	XMLName xml.Name
}

type AcknowledgementMarketDocument struct {
	XMLName                     xml.Name `xml:"Acknowledgement_MarketDocument"`
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
	Reason                                  struct {
		Code int    `xml:"code"`
		Text string `xml:"text"`
	} `xml:"Reason"`
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
		Text        string  `xml:",chardata"`
		Position    int     `xml:"position"`     // 1, 2, 3, 4, 5, 6, 7, 8, 9...
		PriceAmount float64 `xml:"price.amount"` // 16.50, 15.50, 14.00, 10.0...
		Quantity    string  `xml:"quantity"`     // 226, 87, 104, 189, 217, 8...
	} `xml:"Point"`
}
