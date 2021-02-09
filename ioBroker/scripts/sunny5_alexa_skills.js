function alexaSpeak(serial, text) {
    if (text != '') {
        setState('alexa2.0.Echo-Devices.'+serial+'.Commands.speak', text);
        log ("Alexa-Ausgabe: " + text);  
    } else {
        log ("Alexa-Ausgabe: Alexa sollte einen leeren Text sprechen!");
    }
 }


on({ id:'alexa2.0.History.json', change: 'any'}, function(stateObjHistory) {
    let historyObject = JSON.parse(stateObjHistory.state.val);
    var pv = getState("mqtt.0.sunny5.pv_energy").val;
    var connected = getState("mqtt.0.evcc.loadpoints.1.connected").val;
    var charging = getState("mqtt.0.evcc.loadpoints.1.charging").val;
    var chargePower = getState("mqtt.0.evcc.loadpoints.1.chargePower").val;
    var grid = getState("mqtt.0.sunny5.grid").val;
    var consumption = getState("mqtt.0.sunny5.consumption").val;
    var socCharge = getState("mqtt.0.evcc.loadpoints.1.socCharge").val;
    var range = getState("mqtt.0.evcc.loadpoints.1.range").val;
    var mode = getState("mqtt.0.evcc.loadpoints.1.mode").val;
    mode = mode == "now" ? "Sofortladen" : (mode == "minpv" ? "Min+PV laden" : (mode == "pv" ? "PV-Überschussladen" : "STOP"));
 
    if(historyObject.summary == 'was macht die wallbox' || historyObject.summary == 'was macht das elektro auto' || historyObject.summary == 'was macht das elektroauto') {
        log('Alexa: Ausgabe der Wallbox Leistungsdaten.')
        if (connected) {
            var speak = "Dein Elektroauto ist an die Worlbox angeschlossen";
            if (charging) {       
                if (grid <= 0) {
                    speak += "und wird zu 100% mit Sonnenstrom aufgeladen, Ladeleistung " + chargePower + " Watt.";
                }
                else {
                    var autarchy = grid <= 0 ? 100 : (grid >= consumption ? 0 : Math.round((1-(grid/consumption))*100));
                    speak += " und läd mit einer Leistung von "+chargePower+" Watt. Davon sind "+autarchy+"% Sonnenstrom und "+(100-autarchy)+"% Fremdstrom .";
                }

                speak += socCharge > 0 ? ("Der Akkustand beträgt "+socCharge+"% ") : "";
                speak += range > 0 ? ("und die Reichweite "+range+" Kilometer . ") : "";
                speak += "Der Lademodus ist "+mode+" ."; 
            }
            else {
                speak += socCharge > 0 ? " und wird aktuell nicht geladen. Der Ladestand beträgt "+socCharge+"%" : "";
                speak += range > 0 ? " und die Reichweite "+range+" kilometer ." : "";
                speak += " Der Lademodus ist auf "+mode+" eingestellt.";
            }
        }
        else {
            var speak = "Aktuell ist kein Elektroauto mit der Worlbox verbunden.";
        }

        setState('alexa2.0.Echo-Devices.' + historyObject.serialNumber + '.Commands.speak', speak);
    }

    if(historyObject.summary == 'was macht die solaranlage' || historyObject.summary == 'was macht der wechselrichter') {
        log('Alexa: Ausgabe der Solar Leistungsdaten.')
        var soc = getState("mqtt.0.sunny5.soc").val;
        var pv = getState("mqtt.0.sunny5.pv_energy").val;
        if (pv == 0) {
            var speak = "Du erzeugst aktuell keinen Solarstrom. Der Stromverbrauch beträgt " + consumption + "Watt und Dein Batteriespeicher ist zu " + soc + "Prozent geladen.";
        } else {
            var speak = "Deine PV Anlage erzeugt aktuell " + pv + " Watt Solarstrom. Der Stromverbrauch beträgt " + consumption + "Watt und Dein Batteriespeicher ist zu " + soc + "Prozent geladen.";
        }
        setState('alexa2.0.Echo-Devices.' + historyObject.serialNumber + '.Commands.speak', speak);
 
    }
});
