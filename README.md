This is a fork of evcc-io/evcc to add an alternative wake up method for the new Renault E-Tech vehicles (Scenic, 5, and probably 4) to try to solve the issue described [here](https://github.com/evcc-io/evcc/issues/20504)

To use the new wakeup method, the vehicle configuration in evcc.yaml should include, instead of the original "alternativewakeup", the new option "wakeupmode: new"
