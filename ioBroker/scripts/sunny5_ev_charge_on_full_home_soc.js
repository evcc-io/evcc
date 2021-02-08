/*********************************************************************************************
 * Purpose of this Script: Switch from pv charge mode of evcc to min+pv when the home
 * batterie SoC is above 95% and switch back to pv mode when home battery SoC is below 80%.
 * The goal is to prevent high energy states in the home battery for a longer time to
 * optimize battery lifetime.
 * Another important side effect: On hybrid inverters with zero-export to grid it is
 * impossible to measure PV power when home battery is full. The evcc pv charging won't work.
 * This script forces discharge on full home batteries and prevents the above problem.
 *********************************************************************************************/

var isConnected = getState("mqtt.0.evcc.loadpoints.1.connected").val;
var mode = getState("mqtt.0.evcc.loadpoints.1.mode").val;
var soc = getState("mqtt.0.sunny5.soc").val;
var automate = false;

function checkMode() {
    if (isConnected && mode == 'pv' && soc >= 95 && automate == false) {
        log('switch from pv mode to minpv due to home soc: ' + soc + '%');
        automate = true;
        setTimeout(function() { setState("mqtt.0.evcc.loadpoints.1.mode.set", 'minpv') }, 1500);
        return;
    }

    if (mode == 'minpv' && soc <= 80 && automate) {
        log('switch from minpv back to pv mode due to home soc: ' + soc + '%');
        setState("mqtt.0.evcc.loadpoints.1.mode.set", 'pv');
    }
}

on({ id: 'mqtt.0.evcc.loadpoints.1.connected', change:'any'}, (obj) => {
    isConnected = obj.state.val;
    log('ev connected = ' + isConnected);
    checkMode();
})

on({ id: 'mqtt.0.evcc.loadpoints.1.mode.set', change:'any'}, (obj) => {
    log('set mode was confirmed ' + obj.state.val);
    //need this ugly workaround to work in ioBroker
    if (obj.state.val != 'ok') setState("mqtt.0.evcc.loadpoints.1.mode.set", 'ok');
})

on({ id: 'mqtt.0.evcc.loadpoints.1.mode', change:'any'}, (obj) => {
    log('evcc charge mode was set to ' + obj.state.val);
    mode = obj.state.val;

    if (mode != 'minpv') { automate = false; }
    //automate = false;
    checkMode();
})

on({ id: 'mqtt.0.sunny5.soc', change:'any'}, (obj) => {
    soc = obj.state.val;
    checkMode();
})

log('Started at SoC=' + soc + ', mode=' + mode + ', connected=' + isConnected)
checkMode();
