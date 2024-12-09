package stiebel

// Note: all registers are 1-based, i.e. actual value is addr-1

var Block5 = []Register{
	{4001, "SG READY EIN- UND AUSSCHALTEN", "", "", Bits, 0},
	{4002, "SG READY EINGANG 1", "", "", Bits, 0},
	{4003, "SG READY EINGANG 2", "", "", Bits, 0},
}

var Block6 = []Register{
	{5001, "SG READY BETRIEBSZUSTAND", "", "", Uint16, 0},
	{5002, "REGLERKENNUNG", "", "", Uint16, 0},
}
