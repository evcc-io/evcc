package livewire

// Live backend responses captured on 2026-09-11, identifiers anonymized.
const (
	sampleBikes = `{"bikes":[{"id":"100000001","modifiedTime":"2025-10-18 13:12:31+0000","createdTime":"2025-10-18 13:12:31+0000","vin":"7TM3GDYD6SB000000","year":2025,"make":"LiveWire","model":"S2 Mulholland","name":"Mulholland","primaryBike":true,"color":"Liquid Black / Black","colorCode":"M36","modelNumber":"3GDY","modelCode":"S2MH","htmlCodes":["#2A2A2A","#000000"],"brandShortName":"LW"}]}`

	sampleLocationIdle = `{"data":{"vin":"7TM3GDYD6SB000000","iccid":"0","tcu_timestamp":"2026-09-11T16:23:56.000Z","viot_timestamp":"2026-09-11T16:23:58.350Z","latitude":48.1371,"longitude":11.5754,"speed_km":0.0,"altitude":510.5,"gps_time":"2026-09-11T16:22:33.000Z","fix_type":3}}`

	samplePairStatusAfter = `{"bikes":[{"id":"100000001","modifiedTime":"2025-10-18 13:12:31+0000","createdTime":"2025-10-18 13:12:31+0000","vin":"7TM3GDYD6SB000000","year":2025,"make":"LiveWire","model":"S2 Mulholland","name":"Mulholland","primaryBike":true,"color":"Liquid Black / Black","colorCode":"M36","modelNumber":"3GDY","modelCode":"S2MH","pairingStatus":true,"brandShortName":"LW"}]}`

	sampleSession = `{"jwt":"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ4IiwiZGF0YUNlbnRlciI6InVzMSIsImlhdCI6MTc4OTE0MzgwMH0.c2ln","termsAccepted":true}`

	sampleStatusIdle = `{"bikeChargingData":{"batteryPercentage":65,"chargingStatus":false,"durationElapsed":"4","durationUnit":"seconds","maxLimit":80,"odometer":550.632882618,"pluggedIn":false,"range":67,"timeToMaxLimit":"0.0"}}`
)
