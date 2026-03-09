Understand the codebase.

Deeply understand how Site determines available power.
Deeply understand how Loadpoint uses the available power to control a charger using the api.Charger interface.
Deeply understand the difference between available power at the Site level and controlling charger via current and phases.

We want to modify the architecture to support different kinds of chargers, both controlled by power (without directly controlling phases) and controlled by current (the current design).

Ultrathink how this could be supported. Consider making controlling by current a sub type of controlling by power where current control translates power control into currents and phases. Split power and current control into different components. Make the loadpoint operate on a api.PowerController and make the api.CurrentController (with optional phase switching) an implementation of the power controller.

When doing this, reduce the complexity of the loadpoint implementation. Allow chargers to offer either api.PowerController or api.CurrentController.

Develop a plan how to modify the architecture to achieve this. Write it to PLAN.md.
