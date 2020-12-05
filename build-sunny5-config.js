const SubnetsPinger = require('ping-subnet');
const subnetPinger = new SubnetsPinger();
const arp = require('@network-utils/arp-lookup');
const fs = require('fs');
var found = 0;
var round = 1;
var inverterIP, wallboxIP = '';
var arpTable;


subnetPinger.on('host:alive', async ip => {
  arpTable.forEach (item => {
    if (item.ip == ip && item.vendor == 'Shanghai Mxchip Information Tech Co, Ltd') {
      console.log('Found', ip, '\t', item.mac, '\t', 'Sunny5-Hybrid-Inverter');
      inverterIP = ip;
      found++;
    }
    if (item.ip == ip && item.vendor == 'Keba GmbH') {
      console.log('Found', ip, '\t', item.mac, '\t', 'Keba / BMW Charge Plus Wallbox');
      wallboxIP = ip;
      found++
    }
  })
});


subnetPinger.on('ping:end', () => {
  if (found == 2) {
    console.log('');
    console.log('Wallbox and Sunny5-Hybrid-Inverter found, writing new config to ' + process.argv[2] + ' and ' + process.argv[3]);
    
    if (fillTemplateWriteConfig(process.argv[2], process.argv[3], inverterIP, wallboxIP)) {
      console.log('Done, you can run \"./evcc -c ' + process.argv[2] + '\" now.');
    } else {
      console.log('Error using template files', process.argv[2], 'and/or', process.argv[3], '- Check the templates.');
    }
    process.exit(0);
  } 

  if (round < 2 ) {
    round++
    console.log('Not all devices found. Performing second attempt...')
    discover();
  } else {
    console.log('Unable to find all devices. Check hardware and make sure that your Wallbox, Sunny5-Hybrid-Inverter and Sunny5-Broker are in the same network subnet.')
    process.exit(2);
  }

});


function fillTemplateWriteConfig(templateEvcc, templateSunny5, inverterIP, wallboxIP) {
  var template;
  //evcc config file
  try {
    template = fs.readFileSync(templateEvcc, 'utf8');
  }
  catch(err) {
    console.log (err);
    return false
  }
  var re = /(.*uri\:\s)(.*)(\:7090\s\#\swallbox\saddress.*)/;
  var newConfigEvcc = template.replace(re, '$1' + wallboxIP + '$3');
  try {
    fs.writeFileSync(templateEvcc, newConfigEvcc);
  }
  catch(err) {
    console.log (err);
    return false
  }

  //sunny5 config file
  try {
    template = fs.readFileSync(templateSunny5, 'utf8');
  }
  catch(err) {
    console.log (err);
    return false
  }
  re = /(.*host\:\s\')(.*)(\'\,.*)/;  
  var newConfigSunny5 = template.replace(re, '$1' + inverterIP + '$3');
  try {
    fs.writeFileSync(templateSunny5, newConfigSunny5);
  }
  catch(err) {
    console.log (err);
    return false
  }
  
  return true;
}
 
async function discover() {
  arpTable = await arp.getTable()

  subnetPinger.ping();
}

function main() {
  console.log('');
  console.log('****************************************************');
  console.log('*** Sunny5-Broker config file builder, v1.1      ***');
  console.log('****************************************************');
  console.log('');
  
  if (process.argv.length !== 4) { 
    console.log('Invalid command line arguments, specify two config files:');
    console.log('node build-sunny5-config.js evcc.sunny5.yaml ../Sunny5Lib/config.js')
    process.exit(1);
  }

  console.log('Discovering network devices to detect Wallbox and Sunny5-Hybrid-Inverter IPs, please wait a moment ....');
  
  discover();
}

main();
