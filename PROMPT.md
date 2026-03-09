# 1

Understand the codebase.

Deeply understand how Site determines available power.
Deeply understand how Loadpoint uses the available power to control a charger using the api.Charger interface.
Deeply understand the difference between available power at the Site level and controlling charger via current and phases.

We want to modify the architecture to support different kinds of chargers, both controlled by power (without directly controlling phases) and controlled by current (the current design).

Ultrathink how this could be supported. Consider making controlling by current a sub type of controlling by power where current control translates power control into currents and phases. Split power and current control into different components. Make the loadpoint operate on a api.PowerController and make the api.CurrentController (with optional phase switching) an implementation of the power controller.

When doing this, reduce the complexity of the loadpoint implementation. Allow chargers to offer either api.PowerController or api.CurrentController.

Develop a plan how to modify the architecture to achieve this. Write it to PLAN.md.

# 2

It seems you have not created core/chargecontroller/current.go as planned. Instead, now core/loadpoint_controller.go wraps the loadpoint logic for current control.

Due to this, loadpoint.go still contains a lot of current specific logic. This is not what we need. Loadpoint should operate exlcusively on power.

Ultrathink how to extract the current control logic from the loadpoint and move it into core/chargecontroller/current.go. Add this to the plan.

## 2.1

instead "Vehicle overrides via UpdateVehicle" allow the controller to invoke loadpoint methods, but keep the method set minimal and create an interface for it

EffectiveChargePower() on the interface hides Zoe hysteresis and IntegratedDevice special cases
