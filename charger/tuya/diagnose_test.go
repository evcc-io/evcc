package tuya

func ExampleDiagnose() {
	Diagnose(map[string]any{"150": float64(10), "3": "charger_free", "101": float64(200), "999": true}, map[string]string{"3": "work_state", "101": "x_work_state", "150": "x_charge_current"})
	// Output:
	// 	3 work_state:	charger_free
	// 	101 x_work_state:	200
	// 	150 x_charge_current:	10
	// 	999 :	true
}
