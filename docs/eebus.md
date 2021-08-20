# Installation

Based on the installation described in README.me and the wiki, only explaining the differences needed to get chargers supporting EEBUS working.

1. Run `evcc eebus-cert`
2. Add the output to the `evcc.yaml` configuration file
3. Open the web interface of the charger to get the chargers SKI (Identifcation Number)
   For Porsche Mobile Charger Connect this is available in the top menu "Connections" sub-menu "Energy Manager"
4. Add the charger to your configuration:
   ```
   chargers:
   - name: mcc
     type: eebus
     ski: 1234-5678-9012-3456-7890-1234-5678-9012-3456
   ```
5. Run `evcc`
6. On the web interface of the charger typically in the page showing the chargers SKI, `EVCC` should be shown including an option to pair the charger with `EVCC`. Do just that.
7. The EVCC web interface should show the charger and status of a connected car and allow to charge
