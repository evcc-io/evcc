package eebus

import (
	eebusapi "github.com/enbility/eebus-go/api"
	"github.com/enbility/eebus-go/features/client"
	ucapi "github.com/enbility/eebus-go/usecases/api"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/enbility/spine-go/model"
	"github.com/enbility/spine-go/util"
)

// obligation limit filter, matching the OPEV use case (overload protection)
var obligationFilter = model.LoadControlLimitDescriptionDataType{
	LimitType:     util.Ptr(model.LoadControlLimitTypeTypeMaxValueLimit),
	LimitCategory: util.Ptr(model.LoadControlCategoryTypeObligation),
	Unit:          util.Ptr(model.UnitOfMeasurementTypeA),
	ScopeType:     util.Ptr(model.ScopeTypeTypeOverloadProtection),
}

// recommendation limit filter, matching the OSCEV use case (self consumption)
var recommendationFilter = model.LoadControlLimitDescriptionDataType{
	LimitType:     util.Ptr(model.LoadControlLimitTypeTypeMaxValueLimit),
	LimitCategory: util.Ptr(model.LoadControlCategoryTypeRecommendation),
	Unit:          util.Ptr(model.UnitOfMeasurementTypeA),
	ScopeType:     util.Ptr(model.ScopeTypeTypeSelfConsumption),
}

// WriteCombinedLoadControlLimits writes obligation and recommendation limits in a
// single message.
//
// Writing them separately breaks devices that do not support partial writes: each
// write is expanded to the full limit list from the locally cached remote data, so
// the second write reverts the limit ids of the first to their cached state. On a
// disable this leaves all limits inactive, which the EVSE reads as "no limit" and
// charges at maximum.
func WriteCombinedLoadControlLimits(localEntity spineapi.EntityLocalInterface, remoteEntity spineapi.EntityRemoteInterface,
	obligations, recommendations []ucapi.LoadLimitsPhase,
) error {
	loadControl, err := client.NewLoadControl(localEntity, remoteEntity)
	if err != nil {
		return eebusapi.ErrNoCompatibleEntity
	}

	electricalConnection, err := client.NewElectricalConnection(localEntity, remoteEntity)
	if err != nil {
		return eebusapi.ErrNoCompatibleEntity
	}

	data := limitData(loadControl, electricalConnection, obligationFilter, obligations)
	data = append(data, limitData(loadControl, electricalConnection, recommendationFilter, recommendations)...)

	if len(data) == 0 {
		return eebusapi.ErrMissingData
	}

	return Await(func(cb func(model.ResultDataType, model.MsgCounterType)) (*model.MsgCounterType, error) {
		return writeLimitData(loadControl, data, cb)
	})
}

// writeLimitData writes the limit data and registers cb for the write result.
func writeLimitData(loadControl *client.LoadControl, data []model.LoadControlLimitDataType,
	cb func(model.ResultDataType, model.MsgCounterType),
) (*model.MsgCounterType, error) {
	msgCounter, err := loadControl.WriteLimitData(data, nil, nil)
	if err != nil || msgCounter == nil {
		return msgCounter, err
	}

	err = loadControl.AddResponseCallback(*msgCounter, func(msg spineapi.ResponseMessage) {
		if res, ok := msg.Data.(*model.ResultDataType); ok {
			cb(*res, *msgCounter)
		}
	})

	return msgCounter, err
}

// limitData resolves the phase limits matching filter into writeable limit data.
// Phases without a matching, changeable limit id are skipped.
func limitData(loadControl *client.LoadControl, electricalConnection *client.ElectricalConnection,
	filter model.LoadControlLimitDescriptionDataType, limits []ucapi.LoadLimitsPhase,
) []model.LoadControlLimitDataType {
	limitDescriptions, err := loadControl.GetLimitDescriptionsForFilter(filter)
	if err != nil || limitDescriptions == nil {
		return nil
	}

	var res []model.LoadControlLimitDataType

	for _, phaseLimit := range limits {
		// electricalParameterDescription contains the measured phase for each measurementId
		elParamDescs, err := electricalConnection.GetParameterDescriptionsForFilter(
			model.ElectricalConnectionParameterDescriptionDataType{
				AcMeasuredPhases: util.Ptr(phaseLimit.Phase),
			})
		if err != nil || len(elParamDescs) == 0 {
			continue
		}

		// a phase may have several parameter descriptions (e.g. current, voltage, power),
		// so match the one referenced by one of the limit descriptions
		paramDesc, limitDesc := matchDescriptions(elParamDescs, limitDescriptions)
		if paramDesc == nil || limitDesc == nil {
			continue
		}

		limitIdData, err := loadControl.GetLimitDataForId(*limitDesc.LimitId)
		if err != nil {
			continue
		}

		// EEBus_UC_TS_OverloadProtectionByEvChargingCurrentCurtailment V1.01b 3.2.1.2.2.2
		// If omitted or set to "true", the timePeriod, value and isLimitActive element SHALL be writeable by a client.
		if limitIdData.IsLimitChangeable != nil && !*limitIdData.IsLimitChangeable {
			continue
		}

		// electricalPermittedValueSet contains the allowed min, max and the default values per phase
		value := electricalConnection.AdjustValueToBeWithinPermittedValuesForParameterId(
			phaseLimit.Value, *paramDesc.ParameterId)

		res = append(res, model.LoadControlLimitDataType{
			LimitId:       limitDesc.LimitId,
			IsLimitActive: util.Ptr(phaseLimit.IsActive),
			Value:         model.NewScaledNumberType(value),
		})
	}

	return res
}

// matchDescriptions pairs the parameter and limit description sharing a measurement id.
func matchDescriptions(elParamDescs []model.ElectricalConnectionParameterDescriptionDataType,
	limitDescriptions []model.LoadControlLimitDescriptionDataType,
) (*model.ElectricalConnectionParameterDescriptionDataType, *model.LoadControlLimitDescriptionDataType) {
	for _, pd := range elParamDescs {
		if pd.MeasurementId == nil || pd.ParameterId == nil {
			continue
		}

		for _, desc := range limitDescriptions {
			if desc.MeasurementId == nil || desc.LimitId == nil || *desc.MeasurementId != *pd.MeasurementId {
				continue
			}

			return &pd, &desc
		}
	}

	return nil, nil
}
