const SubnetsPinger = require('ping-subnet');
const subnetPinger = new SubnetsPinger();
const arp = require('@network-utils/arp-lookup');
var found = 0;
var arpTable;

subnetPinger.on('host:alive', async ip => {
  arpTable.forEach (item => {
    if (item.ip == ip && item.vendor == 'Shanghai Mxchip Information Tech Co, Ltd') {
      console.log('Found', ip, '\t', item.mac, '\t', 'Sunny5-Hybrid-Inverter');
      found++;
    }
    if (item.ip == ip && item.vendor == 'Keba GmbH') {
      console.log('Found', ip, '\t', item.mac, '\t', 'Keba / BMW Charge Plus Wallbox');
      found++
    }
  })
});
 
subnetPinger.once('ping:end', () => {
  if (found == 2) {
    console.log('');
    console.log('Wallbox and Sunny5 inverter found, writing new config to: evcc.sunny5.yaml');
    console.log('Done, you can run \"./evcc -c evcc.sunny5.yaml\" now.');
  }
});
 
async function discover() {
  arpTable = await arp.getTable()

  subnetPinger.ping();
}

console.log('');
console.log('****************************************************');
console.log('*** Sunny5 Broker config file builder, v0.1      ***');
console.log('****************************************************');
console.log('');
console.log('Discovering network devices to detect Wallbox and Sunny5-Hybrid-Inverter IPs, please wait a moment ....')
discover();
