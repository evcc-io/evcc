package entsoe

import (
	"fmt"
	"strings"
)

var zones = map[string][]string{
	"10Y1001A1001A016": {"CTA|NIE", "MBA|SEM(SONI)", "SCA|NIE"},
	"10Y1001A1001A39I": {"SCA|EE", "MBA|EE", "CTA|EE", "BZN|EE", "Estonia (EE)"},
	"10Y1001A1001A44P": {"IPA|SE1", "BZN|SE1", "MBA|SE1", "SCA|SE1"},
	"10Y1001A1001A45N": {"SCA|SE2", "MBA|SE2", "BZN|SE2", "IPA|SE2"},
	"10Y1001A1001A46L": {"IPA|SE3", "BZN|SE3", "MBA|SE3", "SCA|SE3"},
	"10Y1001A1001A47J": {"SCA|SE4", "MBA|SE4", "BZN|SE4", "IPA|SE4"},
	"10Y1001A1001A48H": {"IPA|NO5", "IBA|NO5", "BZN|NO5", "MBA|NO5", "SCA|NO5"},
	"10Y1001A1001A49F": {"SCA|RU", "MBA|RU", "BZN|RU", "CTA|RU"},
	"10Y1001A1001A50U": {"CTA|RU-KGD", "BZN|RU-KGD", "MBA|RU-KGD", "SCA|RU-KGD"},
	"10Y1001A1001A51S": {"SCA|BY", "MBA|BY", "BZN|BY", "CTA|BY"},
	"10Y1001A1001A59C": {"BZN|IE(SEM)", "MBA|IE(SEM)", "SCA|IE(SEM)", "LFB|IE-NIE", "SNA|Ireland"},
	"10Y1001A1001A63L": {"BZN|DE-AT-LU"},
	"10Y1001A1001A64J": {"BZN|NO1A"},
	"10Y1001A1001A65H": {"Denmark (DK)"},
	"10Y1001A1001A66F": {"BZN|IT-GR"},
	"10Y1001A1001A67D": {"BZN|IT-North-SI"},
	"10Y1001A1001A68B": {"BZN|IT-North-CH"},
	"10Y1001A1001A699": {"BZN|IT-Brindisi", "SCA|IT-Brindisi", "MBA|IT-Z-Brindisi"},
	"10Y1001A1001A70O": {"MBA|IT-Z-Centre-North", "SCA|IT-Centre-North", "BZN|IT-Centre-North"},
	"10Y1001A1001A71M": {"BZN|IT-Centre-South", "SCA|IT-Centre-South", "MBA|IT-Z-Centre-South"},
	"10Y1001A1001A72K": {"MBA|IT-Z-Foggia", "SCA|IT-Foggia", "BZN|IT-Foggia"},
	"10Y1001A1001A73I": {"BZN|IT-North", "SCA|IT-North", "MBA|IT-Z-North"},
	"10Y1001A1001A74G": {"MBA|IT-Z-Sardinia", "SCA|IT-Sardinia", "BZN|IT-Sardinia"},
	"10Y1001A1001A75E": {"BZN|IT-Sicily", "SCA|IT-Sicily", "MBA|IT-Z-Sicily"},
	"10Y1001A1001A76C": {"MBA|IT-Z-Priolo", "SCA|IT-Priolo", "BZN|IT-Priolo"},
	"10Y1001A1001A77A": {"BZN|IT-Rossano", "SCA|IT-Rossano", "MBA|IT-Z-Rossano"},
	"10Y1001A1001A788": {"MBA|IT-Z-South", "SCA|IT-South", "BZN|IT-South"},
	"10Y1001A1001A796": {"CTA|DK"},
	"10Y1001A1001A80L": {"BZN|IT-North-AT"},
	"10Y1001A1001A81J": {"BZN|IT-North-FR"},
	"10Y1001A1001A82H": {"BZN|DE-LU", "IPA|DE-LU", "SCA|DE-LU", "MBA|DE-LU"},
	"10Y1001A1001A83F": {"Germany (DE)"},
	"10Y1001A1001A84D": {"MBA|IT-MACRZONENORTH", "SCA|IT-MACRZONENORTH"},
	"10Y1001A1001A85B": {"SCA|IT-MACRZONESOUTH", "MBA|IT-MACRZONESOUTH"},
	"10Y1001A1001A869": {"SCA|UA-DobTPP", "BZN|UA-DobTPP", "CTA|UA-DobTPP"},
	"10Y1001A1001A877": {"BZN|IT-Malta"},
	"10Y1001A1001A885": {"BZN|IT-SACOAC"},
	"10Y1001A1001A893": {"BZN|IT-SACODC", "SCA|IT-SACODC"},
	"10Y1001A1001A91G": {"SNA|Nordic", "REG|Nordic", "LFB|Nordic"},
	"10Y1001A1001A92E": {"United Kingdom (UK)"},
	"10Y1001A1001A93C": {"Malta (MT)", "BZN|MT", "CTA|MT", "SCA|MT", "MBA|MT"},
	"10Y1001A1001A990": {"MBA|MD", "SCA|MD", "CTA|MD", "BZN|MD", "Moldova (MD)"},
	"10Y1001A1001B004": {"Armenia (AM)", "BZN|AM", "CTA|AM"},
	"10Y1001A1001B012": {"CTA|GE", "BZN|GE", "Georgia (GE)", "SCA|GE", "MBA|GE"},
	"10Y1001A1001B05V": {"Azerbaijan (AZ)", "BZN|AZ", "CTA|AZ"},
	"10Y1001C--00003F": {"BZN|UA", "Ukraine (UA)", "MBA|UA", "SCA|UA"},
	"10Y1001C--000182": {"SCA|UA-IPS", "MBA|UA-IPS", "BZN|UA-IPS", "CTA|UA-IPS"},
	"10Y1001C--00038X": {"BZA|CZ-DE-SK-LT-SE4"},
	"10Y1001C--00059P": {"REG|CORE"},
	"10Y1001C--00090V": {"REG|AFRR", "SCA|AFRR"},
	"10Y1001C--00095L": {"REG|SWE"},
	"10Y1001C--00096J": {"SCA|IT-Calabria", "MBA|IT-Z-Calabria", "BZN|IT-Calabria"},
	"10Y1001C--00098F": {"BZN|GB(IFA)"},
	"10Y1001C--00100H": {"BZN|XK", "CTA|XK", "Kosovo (XK)", "MBA|XK", "LFB|XK", "LFA|XK"},
	"10Y1001C--00119X": {"SCA|IN"},
	"10Y1001C--001219": {"BZN|NO2A"},
	"10Y1001C--00137V": {"REG|ITALYNORTH"},
	"10Y1001C--00138T": {"REG|GRIT"},
	"10YAL-KESH-----5": {"LFB|AL", "LFA|AL", "BZN|AL", "CTA|AL", "Albania (AL)", "SCA|AL", "MBA|AL"},
	"10YAT-APG------L": {"MBA|AT", "SCA|AT", "Austria (AT)", "IPA|AT", "CTA|AT", "BZN|AT", "LFA|AT", "LFB|AT"},
	"10YBA-JPCC-----D": {"LFA|BA", "BZN|BA", "CTA|BA", "Bosnia and Herz. (BA)", "SCA|BA", "MBA|BA"},
	"10YBE----------2": {"MBA|BE", "SCA|BE", "Belgium (BE)", "CTA|BE", "BZN|BE", "LFA|BE", "LFB|BE"},
	"10YCA-BULGARIA-R": {"LFB|BG", "LFA|BG", "BZN|BG", "CTA|BG", "Bulgaria (BG)", "SCA|BG", "MBA|BG"},
	"10YCB-GERMANY--8": {"SCA|DE_DK1_LU", "LFB|DE_DK1_LU"},
	"10YCB-JIEL-----9": {"LFB|RS_MK_ME"},
	"10YCB-POLAND---Z": {"LFB|PL"},
	"10YCB-SI-HR-BA-3": {"LFB|SI_HR_BA"},
	"10YCH-SWISSGRIDZ": {"LFB|CH", "LFA|CH", "SCA|CH", "MBA|CH", "Switzerland (CH)", "CTA|CH", "BZN|CH"},
	"10YCS-CG-TSO---S": {"BZN|ME", "CTA|ME", "Montenegro (ME)", "MBA|ME", "SCA|ME", "LFA|ME"},
	"10YCS-SERBIATSOV": {"LFA|RS", "SCA|RS", "MBA|RS", "Serbia (RS)", "CTA|RS", "BZN|RS"},
	"10YCY-1001A0003J": {"BZN|CY", "CTA|CY", "Cyprus (CY)", "MBA|CY", "SCA|CY"},
	"10YCZ-CEPS-----N": {"SCA|CZ", "MBA|CZ", "Czech Republic (CZ)", "CTA|CZ", "BZN|CZ", "LFA|CZ", "LFB|CZ"},
	"10YDE-ENBW-----N": {"LFA|DE(TransnetBW)", "CTA|DE(TransnetBW)", "SCA|DE(TransnetBW)"},
	"10YDE-EON------1": {"SCA|DE(TenneT GER)", "CTA|DE(TenneT GER)", "LFA|DE(TenneT GER)"},
	"10YDE-RWENET---I": {"LFA|DE(Amprion)", "CTA|DE(Amprion)", "SCA|DE(Amprion)"},
	"10YDE-VE-------2": {"SCA|DE(50Hertz)", "CTA|DE(50Hertz)", "LFA|DE(50Hertz)", "BZA|DE(50HzT)"},
	"10YDK-1-------AA": {"BZN|DK1A"},
	"10YDK-1--------W": {"IPA|DK1", "IBA|DK1", "BZN|DK1", "SCA|DK1", "MBA|DK1", "LFA|DK1"},
	"10YDK-2--------M": {"LFA|DK2", "MBA|DK2", "SCA|DK2", "IBA|DK2", "IPA|DK2", "BZN|DK2"},
	"10YDOM-1001A082L": {"CTA|PL-CZ", "BZA|PL-CZ"},
	"10YDOM-CZ-DE-SKK": {"BZA|CZ-DE-SK", "BZN|CZ+DE+SK"},
	"10YDOM-PL-SE-LT2": {"BZA|LT-SE4"},
	"10YDOM-REGION-1V": {"REG|CWE"},
	"10YES-REE------0": {"LFB|ES", "LFA|ES", "BZN|ES", "Spain (ES)", "CTA|ES", "SCA|ES", "MBA|ES"},
	"10YEU-CONT-SYNC0": {"SNA|Continental Europe"},
	"10YFI-1--------U": {"MBA|FI", "SCA|FI", "CTA|FI", "Finland (FI)", "BZN|FI", "IPA|FI", "IBA|FI"},
	"10YFR-RTE------C": {"BZN|FR", "France (FR)", "CTA|FR", "SCA|FR", "MBA|FR", "LFB|FR", "LFA|FR"},
	"10YGB----------A": {"LFA|GB", "LFB|GB", "SNA|GB", "MBA|GB", "SCA|GB", "CTA|National Grid", "BZN|GB"},
	"10YGR-HTSO-----Y": {"BZN|GR", "Greece (GR)", "CTA|GR", "SCA|GR", "MBA|GR", "LFB|GR", "LFA|GR"},
	"10YHR-HEP------M": {"LFA|HR", "MBA|HR", "SCA|HR", "CTA|HR", "Croatia (HR)", "BZN|HR"},
	"10YHU-MAVIR----U": {"BZN|HU", "Hungary (HU)", "CTA|HU", "SCA|HU", "MBA|HU", "LFA|HU", "LFB|HU"},
	"10YIE-1001A00010": {"MBA|SEM(EirGrid)", "SCA|IE", "CTA|IE", "Ireland (IE)"},
	"10YIT-GRTN-----B": {"Italy (IT)", "CTA|IT", "SCA|IT", "MBA|IT", "LFB|IT", "LFA|IT"},
	"10YLT-1001A0008Q": {"MBA|LT", "SCA|LT", "CTA|LT", "Lithuania (LT)", "BZN|LT"},
	"10YLU-CEGEDEL-NQ": {"Luxembourg (LU)", "CTA|LU"},
	"10YLV-1001A00074": {"CTA|LV", "Latvia (LV)", "BZN|LV", "SCA|LV", "MBA|LV"},
	"10YMK-MEPSO----8": {"MBA|MK", "SCA|MK", "BZN|MK", "North Macedonia (MK)", "CTA|MK", "LFA|MK"},
	"10YNL----------L": {"LFA|NL", "LFB|NL", "CTA|NL", "Netherlands (NL)", "BZN|NL", "SCA|NL", "MBA|NL"},
	"10YNO-0--------C": {"MBA|NO", "SCA|NO", "Norway (NO)", "CTA|NO"},
	"10YNO-1--------2": {"BZN|NO1", "IBA|NO1", "IPA|NO1", "SCA|NO1", "MBA|NO1"},
	"10YNO-2--------T": {"MBA|NO2", "SCA|NO2", "IPA|NO2", "IBA|NO2", "BZN|NO2"},
	"10YNO-3--------J": {"BZN|NO3", "IBA|NO3", "IPA|NO3", "SCA|NO3", "MBA|NO3"},
	"10YNO-4--------9": {"MBA|NO4", "SCA|NO4", "IPA|NO4", "IBA|NO4", "BZN|NO4"},
	"10YPL-AREA-----S": {"BZN|PL", "Poland (PL)", "CTA|PL", "SCA|PL", "MBA|PL", "BZA|PL", "LFA|PL"},
	"10YPT-REN------W": {"LFA|PT", "LFB|PT", "MBA|PT", "SCA|PT", "CTA|PT", "Portugal (PT)", "BZN|PT"},
	"10YRO-TEL------P": {"BZN|RO", "Romania (RO)", "CTA|RO", "SCA|RO", "MBA|RO", "LFB|RO", "LFA|RO"},
	"10YSE-1--------K": {"MBA|SE", "SCA|SE", "CTA|SE", "Sweden (SE)"},
	"10YSI-EELS-----O": {"Slovenia (SI)", "BZN|SI", "CTA|SI", "SCA|SI", "MBA|SI", "LFA|SI"},
	"10YSK-SEPS-----K": {"LFA|SK", "LFB|SK", "MBA|SK", "SCA|SK", "CTA|SK", "BZN|SK", "Slovakia (SK)"},
	"10YTR-TEIAS----W": {"Turkey (TR)", "BZN|TR", "CTA|TR", "SCA|TR", "MBA|TR", "LFB|TR", "LFA|TR"},
	"10YUA-WEPS-----0": {"LFA|UA-BEI", "LFB|UA-BEI", "MBA|UA-BEI", "SCA|UA-BEI", "CTA|UA-BEI", "BZN|UA-BEI"},
	"11Y0-0000-0265-K": {"BZN|GB(ElecLink)"},
	"17Y0000009369493": {"BZN|GB(IFA2)"},
	"46Y000000000007M": {"BZN|DK1-NO1"},
	"50Y0JVU59B4JWQCU": {"BZN|NO2NSL"},
	"BY":               {"Belarus (BY)"},
	"RU":               {"Russia (RU)"},
	"IS":               {"Iceland (IS)"},
}

type AreaType string

const (
	BZN AreaType = "Bidding Zone"
	BZA AreaType = "Bidding Zone Aggregation"
	CTA AreaType = "Control Area"
	MBA AreaType = "Market Balance Area"
	IBA AreaType = "Imbalance Area"
	IPA AreaType = "Imbalance Price Area"
	LFA AreaType = "Load Frequency Control Area"
	LFB AreaType = "Load Frequency Control Block"
	REG AreaType = "Region"
	SCA AreaType = "Scheduling Area"
	SNA AreaType = "Synchronous Area"
)

func Area(typ AreaType, name string) (string, error) {
	combined := fmt.Sprintf("%s|%s", typ, name)

	// allows matching country codes
	suffix := fmt.Sprintf(" (%s)", name)

	for code, names := range zones {
		if code == name {
			return code, nil
		}

		for _, n := range names {
			if n == name || n == combined || strings.HasSuffix(n, suffix) {
				return code, nil
			}
		}
	}
	return "", fmt.Errorf("unknown area: %s", name)
}
