/*****************************************************************************************************
 * Purpose of this script: Battery AC charging
 * This script can be applied when having two pv systems, one is used for self consumption,
 * (sunny5 system) and the other for on-grid (eeg system). On weak days we add ac charging to Sunny5.
 * We determine the solarNoon and check if battery is at least 30% soc, 2 hours before
 * solarNoon, or 40% one hour before, or 50% at solarNoon or 60% one hour after
 * solarNoon. If soc is not reached, AC charge is triggered.
 * If household consumption exceeds 500W we stop AC charge temporarily. to power our loads with green
 * solar and battery power.
 * This helps us to store enough power during the winter times. The "extra AC power" is 
 * taken from our other grid-tie-inverters (solar energy).
 *****************************************************************************************************/

var suncalc = require('suncalc');
 
var times = suncalc.getTimes(new Date(),51.067950,6.197830);
var soc = getState("mqtt.0.sunny5.soc").val;
var bolACEnabled = getState("mqtt.0.sunny5.ac_charge_enable").val;
var consumption = getState("mqtt.0.sunny5.consumption").val;

on({ id: 'mqtt.0.sunny5.ac_charge_enable', change:'any'}, (obj) => {
    times = suncalc.getTimes(new Date(),51.067950,6.197830);
    log('ac state = ' + obj.state.val);
    bolACEnabled = obj.state.val;
})

on({ id: 'mqtt.0.sunny5.soc', change:'any'}, (obj) => {
    times = suncalc.getTimes(new Date(),51.067950,6.197830);
    soc = obj.state.val;
    log('Current soc: ' + soc + '% # ' + bolACEnabled + ' # ' + consumption + ' # ' + (new Date()) + ' # ' + times['solarNoon']);
})

on({ id: 'mqtt.0.sunny5.consumption', change:'any'}, (obj) => {
    let snEpoch = times['solarNoon'].getTime();
    let nowEpoch = (new Date()).getTime();
    consumption = obj.state.val;

    // *** turn on conditions
    //check if 2hours before solarNoon and 30% soc not reached and consumption <= 500W
    if (!bolACEnabled && consumption <= 500 && soc < 30 && nowEpoch >= snEpoch-2*60*60*1000 && nowEpoch < snEpoch-1*60*60*1000) {
        log('2 hours before solarNoon and below 30% soc, turning on ac charge now!')
        setState("mqtt.0.sunny5.ac_charge_enable.set", '1');
        bolACEnabled = true;
        return;
    }

    //check if 1hour before solarNoon and 40% soc not reached
    if (!bolACEnabled && consumption <= 500 && soc < 40 && nowEpoch >= snEpoch-1*60*60*1000 && nowEpoch < snEpoch) {
        log('1 hour before solarNoon and below 40% soc, turning on ac charge now!')
        setState("mqtt.0.sunny5.ac_charge_enable.set", '1');
        bolACEnabled = true;
        return;
    }

    //check if solarNoon and 50% soc not reached
    if (!bolACEnabled && consumption <= 500 && soc < 50 && nowEpoch >= snEpoch && nowEpoch < snEpoch+1*60*60*1000) {
        log('solarNoon and below 50% soc, turning on ac charge now!')
        setState("mqtt.0.sunny5.ac_charge_enable.set", '1');
        bolACEnabled = true;
        return;
    }

    //check if 1hour behind solarNoon and 60% soc not reached
    if (!bolACEnabled && consumption <= 500 && soc < 60 && nowEpoch >= snEpoch+1*60*60*1000 && nowEpoch < snEpoch+2*60*60*1000) {
        log('1 hour behind solarNoon and below 60% soc, turning on ac charge now!')
        setState("mqtt.0.sunny5.ac_charge_enable.set", '1');
        bolACEnabled = true;
        return;
    }

    // *** turn off conditions
    //check if 2hours behind solarNoon
    if (nowEpoch > snEpoch+2*60*60*1000) {
        if (bolACEnabled) {
            log('2 hours behind solarNoon, turning off ac charge now!')
            setState("mqtt.0.sunny5.ac_charge_enable.set", '0');
            bolACEnabled = false;
        }
        return;
    }

    //check if soc has reached 70%
    if (bolACEnabled && soc >= 70) {
        log('SoC is above 70%, turning off ac charge now!')
        setState("mqtt.0.sunny5.ac_charge_enable.set", '0');
        bolACEnabled = false;
    }

    //check if current consumption is above 500W
    if (bolACEnabled && consumption > 500) {
        log('Current household consumption is above 500W, turning off ac charge now!')
        setState("mqtt.0.sunny5.ac_charge_enable.set", '0');
        bolACEnabled = false;
    }
})