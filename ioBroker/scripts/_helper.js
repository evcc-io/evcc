// *********************************
// *** Create new DPs
createState("sunny5.energy.ev_charging_without_grid", false, false, {
    name: "EV charge state - without grid",
    desc: "",
    type: "boolean",
    role: "value",
    unit: ""
 });
 
 createState("sunny5.energy.ev_charging_with_grid", false, false, {
    name: "EV charge state - wit grid",
    desc: "",
    type: "boolean",
    role: "value",
    unit: ""
 });
 
 
 // *********************************
 // *** Event Listener
 var isCharging = getState("mqtt.0.evcc.loadpoints.1.charging").val;
 var grid = getState("mqtt.0.sunny5.grid").val;
 
 on({ id: 'mqtt.0.evcc.loadpoints.1.charging', change:'any'}, (obj) => {
     isCharging = obj.state.val;
     checkChargingStates();
 })
 
 on({ id: 'mqtt.0.sunny5.grid', change:'any'}, (obj) => {
     grid = obj.state.val;
     checkChargingStates();
 })
 
 
 // *********************************
 // *** Set DP states on events
 function checkChargingStates() {
     if (isCharging) {
         if (grid <= 20) {
             setState("sunny5.energy.ev_charging_without_grid", true);
             setState("sunny5.energy.ev_charging_with_grid", false);    
         }
         else {
             setState("sunny5.energy.ev_charging_without_grid", false);
             setState("sunny5.energy.ev_charging_with_grid", true);    
         }
     }
     else {
         setState("sunny5.energy.ev_charging_with_grid", false);
         setState("sunny5.energy.ev_charging_without_grid", false);
     }
 }